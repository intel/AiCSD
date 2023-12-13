/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package functions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"aicsd/as-file-receiver-oem/clients/file_sender"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos"
	"github.com/hashicorp/go-multierror"
)

// PipelineReceiver provides the data for the App Function Pipelines in this package
type PipelineReceiver struct {
	job              types.Job
	fileHostname     string
	outputFolder     string
	lc               logger.LoggingClient
	jobRepoClient    job_repo.Client
	fileSenderClient file_sender.Client
}

func NewPipelineReceiver(jobRepoClient job_repo.Client, fileSenderClient file_sender.Client, fileHostname, outputFolder string) *PipelineReceiver {
	return &PipelineReceiver{
		fileHostname:     fileHostname,
		outputFolder:     outputFolder,
		jobRepoClient:    jobRepoClient,
		fileSenderClient: fileSenderClient,
	}
}

// ProcessEvent is the entry point App Pipeline Function that receives and processes the EdgeX Event/Reading that
// contains the Pipeline Parameters
func (p *PipelineReceiver) ProcessEvent(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc = ctx.LoggingClient()
	p.lc.Debugf("running ProcessEvent...")

	if data == nil {
		return false, pkg.ErrEmptyInput
	}

	event, ok := data.(dtos.Event)
	if !ok {
		p.lc.Error("ProcessEvent failed: type received is not an event")
		return false, werrors.WrapMsg(pkg.ErrInvalidInput, "type received is not an event")
	}

	err := helpers.AppFunctionEventValidation(event, pkg.OwnerFileSenderGateway, pkg.ResourceNameJob)
	if err != nil {
		p.lc.Error("ProcessEvent AppFunctionEventValidation failed: %s", err.Error())
		return false, werrors.WrapMsg(err, "failed AppFunctionEventValidation")
	}

	jsonData, err := json.Marshal(event.Readings[0].ObjectValue)
	if err != nil {
		p.lc.Error("ProcessEvent marshal failed: %s", err.Error())
		return false, werrors.WrapMsg(err, "unable to marshal Object Value back to JSON")
	}

	if err = json.Unmarshal(jsonData, &p.job); err != nil {
		p.lc.Error("ProcessEvent unmarshal failed: %s", err.Error())
		return false, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
	}

	ctx.LoggingClient().Debugf("received the following job: input = %s, output = %s", p.job.FullInputFileLocation(), p.job.FullOutputFileLocation())

	return true, nil // All is good, this indicates success for the next function
}

// UpdateJobRepoOwner is an App Pipeline Function that does the PUT request to update the Job Repo with the owner update.
// Once a Job has been received on the message bus, then ownership will be set to the File Receiver OEM.
func (p *PipelineReceiver) UpdateJobRepoOwner(_ interfaces.AppFunctionContext, _ interface{}) (bool, interface{}) {
	p.lc.Debugf("running UpdateJobRepoOwner for %s", p.job.FullOutputFileLocation())

	var err error
	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerFileRecvOem
	p.job, err = p.jobRepoClient.Update(p.job.Id, jobFields)
	if err != nil {
		p.lc.Errorf("UpdateJobRepoOwner failed: %s", err.Error())
		return false, werrors.WrapErr(err, pkg.ErrUpdating)
	}

	return true, nil // All is good, this indicates success for the next function
}

func (p *PipelineReceiver) handleJobFiles(fileErrChan chan error, wg *sync.WaitGroup, fileId int) {
	defer wg.Done()
	outputFile := p.job.PipelineDetails.OutputFiles[fileId]
	fileName := filepath.Join(p.outputFolder, filepath.Base(outputFile.Name))
	ext := filepath.Ext(fileName)
	p.job.UpdateOutputFile(fileId, p.outputFolder, fileName, ext, "", "", nil, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem)
	fileBytes, transmitErr := p.fileSenderClient.TransmitFile(p.job.Id, strconv.FormatInt(int64(fileId), 10))
	if transmitErr != nil {
		p.lc.Debugf("transmitFile transmit err for file: %s", fileName)
		p.job.UpdateOutputFile(fileId, p.outputFolder, fileName, ext, "", "", pkg.ErrFileTransmitting, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem)
		fileErrChan <- pkg.ErrFileTransmitting
		return
	} else if fileBytes == nil {
		p.lc.Debugf("transmitFile file bytes nil for file: %s", fileName)
		p.job.UpdateOutputFile(fileId, p.outputFolder, fileName, ext, "", "", pkg.ErrFileTransmitting, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem)
		fileErrChan <- pkg.ErrFileTransmitting
		return
	} else {
		p.lc.Debugf("PullFile: writing output file: %s", fileName)
		writeErr := os.WriteFile(fileName, fileBytes, pkg.FilePermissions)
		if writeErr != nil {
			p.lc.Debugf("WriteFile failed writing output file: %s", fileName)
			p.job.UpdateOutputFile(fileId, p.outputFolder, fileName, ext, "", "", pkg.ErrFileWrite, pkg.FileStatusWriteFailed, pkg.OwnerFileRecvOem)
			fileErrChan <- pkg.ErrFileWrite
		}
		return
	}
}

// PullFile is an App Pipeline Function that does the GET request to File Sender GW for files via the TransmitFile API
func (p *PipelineReceiver) PullFile(_ interfaces.AppFunctionContext, _ interface{}) (bool, interface{}) {
	p.lc.Debugf("running PullFile for %s", p.job.FullOutputFileLocation())

	errsChan := make(chan error)
	wgDone := make(chan bool, 1)
	wgForFiles := sync.WaitGroup{}
	var errors error

	go func() {
		wgForFiles.Wait()
		close(wgDone)
	}()

	for key, file := range p.job.PipelineDetails.OutputFiles {
		p.lc.Debugf("processing output file %s for job", file)
		if file.Status == pkg.FileStatusWriteFailed || file.Status == pkg.FileStatusTransmissionFailed || file.Status == pkg.FileStatusInvalid {
			continue
		}
		wgForFiles.Add(1)
		go p.handleJobFiles(errsChan, &wgForFiles, key)
	}

	// wait until WaitGroup is done processing files
	for {
		select {
		case errFromChan := <-errsChan:
			p.lc.Debugf("this is err from job err channel processing job files %s", errFromChan)
			errors = multierror.Append(errors, errFromChan)
			p.job.Owner = pkg.OwnerFileRecvOem
			p.job.Status = pkg.StatusFileError
			p.job.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, errFromChan)
			continue
		case <-wgDone:
			p.lc.Debugf("wgDone")
			if errors != nil {
				return true, errors // All is good, this indicates success for the next function
			}
			return true, nil
		}
	}
}

// ArchiveFile is an App Pipeline Function that does the POST request to File Sender GW to archive files via the ArchiveFile API
func (p *PipelineReceiver) ArchiveFile(_ interfaces.AppFunctionContext, priorErr interface{}) (bool, interface{}) {
	p.lc.Debugf("running ArchiveFile...")

	// Previous function returns any error as its result which is passed to this function.
	// priorErr will be a file err specific to the job.
	var errors error
	if pipelineErr, ok := priorErr.(error); ok {
		p.lc.Info("ArchiveFile: captured error %s from prior function and will archive the files that it can", pipelineErr)
		errors = multierror.Append(errors, pipelineErr)
	}

	err := p.fileSenderClient.ArchiveFile(p.job.Id)
	if err != nil {
		// don't fail if the archive fails, just log it and move on
		p.lc.Errorf("ArchiveFile failed for input file %s, output file %s: %s", p.job.FullInputFileLocation(), p.job.FullOutputFileLocation(), err.Error())
		errors = werrors.WrapErr(err, pkg.ErrFileArchiving)
		p.job.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileArchiving)
	}

	return true, errors // All is good, this indicates success for the next function
}

// UpdateJobRepoComplete is an App Pipeline Function that does the PUT request to update the Job Repo with the owner update.
// Once a file has been received, then the ownership will be set to none.
func (p *PipelineReceiver) UpdateJobRepoComplete(_ interfaces.AppFunctionContext, priorErr interface{}) (bool, interface{}) {
	p.lc.Debugf("running UpdateJobRepoComplete for %s", p.job.FullOutputFileLocation())

	// Previous function returns any error as its result which is passed to this function.
	var errors error
	if pipelineErr, ok := priorErr.(error); ok {
		p.lc.Info("UpdateJobRepoComplete: captured error %s from prior function and will update job", pipelineErr)
		errors = multierror.Append(errors, pipelineErr)
	}

	var err error
	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerNone
	jobFields[types.JobStatus] = pkg.StatusComplete
	// do not overwrite failure status
	if p.job.Status == pkg.StatusFileError {
		jobFields[types.JobOwner] = pkg.OwnerNone
		jobFields[types.JobStatus] = p.job.Status
	}
	jobFields[types.JobPipelineOutputHost] = p.fileHostname
	if p.job.ErrorDetails != nil {
		jobFields[types.JobErrorDetailsOwner] = p.job.ErrorDetails.Owner
		jobFields[types.JobErrorDetailsErrorMsg] = p.job.ErrorDetails.Error
	}

	p.job, err = p.jobRepoClient.Update(p.job.Id, jobFields)
	if err != nil {
		p.lc.Error("UpdateJobRepoComplete failed: %s", err.Error())
		errors = multierror.Append(errors, werrors.WrapErr(err, pkg.ErrUpdating))
	}

	// be sure to return prior function errors if there was one, along with update err if occurs
	return false, errors // All done
}
