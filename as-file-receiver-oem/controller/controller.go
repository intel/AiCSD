/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/as-file-receiver-oem/clients/file_sender"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"
	"aicsd/pkg/werrors"
	"github.com/hashicorp/go-multierror"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
)

type Controller struct {
	lc                logger.LoggingClient
	fileHostname      string
	outputFolder      string
	jobRepoClient     job_repo.Client
	fileSenderClient  file_sender.Client
	DependentServices wait.Services
}

func New(lc logger.LoggingClient, jobRepoClient job_repo.Client, fileSenderClient file_sender.Client, fileHostname, outputFolder string, dependentServices wait.Services) *Controller {
	return &Controller{
		lc:                lc,
		fileHostname:      fileHostname,
		outputFolder:      outputFolder,
		jobRepoClient:     jobRepoClient,
		fileSenderClient:  fileSenderClient,
		DependentServices: dependentServices,
	}
}

func (c *Controller) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointRetry, c.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	return nil
}

// handleJobFiles is a helper function to be called per job to transmit all files from the file sender gateway.
// This function returns errors on a channel.
func (c *Controller) handleJobFiles(fileErrChan chan error, wg *sync.WaitGroup, job *types.Job, fileId int) {
	defer wg.Done()
	outputFile := job.PipelineDetails.OutputFiles[fileId]
	fileName := filepath.Join(c.outputFolder, filepath.Base(outputFile.Name))
	ext := filepath.Ext(fileName)
	job.UpdateOutputFile(fileId, c.outputFolder, fileName, ext, "", "", nil, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem)
	fileBytes, transmitErr := c.fileSenderClient.TransmitFile(job.Id, strconv.FormatInt(int64(fileId), 10))
	if transmitErr != nil {
		c.lc.Debugf("transmitFile transmit err for file: %s", fileName)
		job.UpdateOutputFile(fileId, c.outputFolder, fileName, ext, "", "", pkg.ErrFileTransmitting, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem)
		fileErrChan <- pkg.ErrFileTransmitting
		return

	} else if fileBytes == nil {
		c.lc.Debugf("transmitFile file bytes nil for file: %s", fileName)
		job.UpdateOutputFile(fileId, c.outputFolder, fileName, ext, "", "", pkg.ErrFileTransmitting, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem)
		fileErrChan <- pkg.ErrFileTransmitting
		return
	} else {
		c.lc.Debugf("PullFile: writing output file: %s", fileName)
		writeErr := os.WriteFile(fileName, fileBytes, pkg.FilePermissions)
		if writeErr != nil {
			c.lc.Debugf("WriteFile failed writing output file: %s", fileName)
			job.UpdateOutputFile(fileId, c.outputFolder, fileName, ext, "", "", pkg.ErrFileWrite, pkg.FileStatusWriteFailed, pkg.OwnerFileRecvOem)
			fileErrChan <- pkg.ErrFileWrite
		}
		return
	}
}

// RetryOnStartup will be called on startup to look at what job objects the file receiver oem owns and attempts to
// process them. The function checks for jobs it is owner of and attempts to transfer them.
// It leverages the Go concurrency primitives to create a go routine per job and per output file to transmit the files
// capturing errors in a channel upon file error.
func (c *Controller) RetryOnStartup() error {
	// TODO: add retry logic around the Retrieve call in case job repo is not up yet.
	jobs, err := c.jobRepoClient.RetrieveAllByOwner(pkg.OwnerFileRecvOem)
	if err != nil {
		return werrors.WrapErr(err, pkg.ErrRetrieving)
	}

	var errors error

	// NOTE: This retry call is needed to refresh the jobs on the file-sender-gw
	retryErr := c.fileSenderClient.Retry()
	if retryErr != nil {
		errors = multierror.Append(retryErr, err)
	}

	for _, job := range jobs {
		jobFields := make(map[string]interface{})
		jobsErrChan := make(chan error)
		wgForFiles := sync.WaitGroup{}
		wgDone := make(chan bool, 1)
		go func() {
			wgForFiles.Wait()
			close(wgDone)
		}()
		for key, file := range job.PipelineDetails.OutputFiles {
			c.lc.Debugf("processing output file %s for job", file)
			if file.Status == pkg.FileStatusWriteFailed || file.Status == pkg.FileStatusTransmissionFailed || file.Status == pkg.FileStatusInvalid {
				continue
			}
			wgForFiles.Add(1)
			go c.handleJobFiles(jobsErrChan, &wgForFiles, &job, key)
		}
		for {
			select {
			case jobErr := <-jobsErrChan:
				c.lc.Debugf("this is err from job err channel %s", jobErr)
				errors = multierror.Append(errors, jobErr)
				job.Owner = pkg.OwnerFileRecvOem
				job.Status = pkg.StatusFileError
				job.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, jobErr)
				break
			case <-wgDone:
				c.lc.Debugf("wgDone")
				// archive files if files were transferred successfully
				archiveErr := c.fileSenderClient.ArchiveFile(job.Id)
				if archiveErr != nil {
					// don't fail if the archive fails, just log it and move on
					errors = multierror.Append(werrors.WrapErr(archiveErr, pkg.ErrFileArchiving))
					job.Owner = pkg.OwnerNone
					job.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileArchiving)
					job.Status = pkg.StatusFileError
					c.lc.Errorf("ArchiveFile failed for input file %s, output file %s: %s", job.FullInputFileLocation(), job.FullOutputFileLocation(), archiveErr.Error())
				}
				// do not overwrite failure status
				if job.Status != pkg.StatusFileError {
					job.Owner = pkg.OwnerNone
					job.Status = pkg.StatusComplete
				}
				jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
				jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				jobFields[types.JobPipelineOutputHost] = c.fileHostname
				jobFields[types.JobOwner] = job.Owner
				jobFields[types.JobStatus] = job.Status
				c.lc.Debugf("updating job id %s with output files %s", job.Id, job.PipelineDetails.OutputFiles)
				_, updateErr := c.jobRepoClient.Update(job.Id, jobFields)
				if updateErr != nil {
					errors = multierror.Append(werrors.WrapErr(updateErr, pkg.ErrUpdating))
				}
				goto NEXTJOB
			}
		}
	NEXTJOB:
	}
	return errors

}

// The retry function is a wrapper for the RetryOnStartup call that is utilized by the retry endpoint
func (c *Controller) retry(writer http.ResponseWriter, request *http.Request) {
	err := c.RetryOnStartup()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	c.lc.Debug("Retry endpoint successfully called")
}
