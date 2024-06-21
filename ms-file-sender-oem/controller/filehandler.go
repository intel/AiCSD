/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/ms-file-sender-oem/clients/file_receiver"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"
	"aicsd/pkg/werrors"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/hashicorp/go-multierror"
)

type FileHandler struct {
	lc                 logger.LoggingClient
	jobRepoClient      job_repo.Client
	fileReceiverClient file_receiver.Client
	fileHostname       string
	DependentServices  wait.Services
}

func New(lc logger.LoggingClient, jobRepoClient job_repo.Client, fileReceiverClient file_receiver.Client, fileHostname string, dependentServices wait.Services) *FileHandler {
	return &FileHandler{
		lc:                 lc,
		jobRepoClient:      jobRepoClient,
		fileReceiverClient: fileReceiverClient,
		fileHostname:       fileHostname,
		DependentServices:  dependentServices,
	}
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred
func (fh *FileHandler) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointDataToHandle, fh.SendNewFile, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointDataToHandle)
	}
	err = service.AddRoute(pkg.EndpointRetry, fh.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	return nil
}

// RetryOnStartup gets all job entries that the file-sender-oem owns, and for each entry
// attempts to transmit the job and the file
func (fh *FileHandler) RetryOnStartup() error {
	var err, errs error
	jobs, err := fh.jobRepoClient.RetrieveAllByOwner(pkg.OwnerFileSenderOem)
	if err != nil {
		return fmt.Errorf("could not retrieve %s data: %s", pkg.OwnerFileSenderOem, err.Error())
	}
	for _, currentJob := range jobs {
		fh.lc.Debugf("retrying Job with input file %s", currentJob.FullInputFileLocation())
		err = fh.fileReceiverClient.TransmitJob(currentJob)
		if err != nil {
			jobFields := make(map[string]interface{})
			if !helpers.IsNetworkError(err) {
				jobFields[types.JobStatus] = pkg.StatusTransmissionFailed
			} else {
				jobFields[types.JobStatus] = pkg.StatusIncomplete
			}

			jobFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderOem
			jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrTransmitJob.Error()
			_, updateErr := fh.jobRepoClient.Update(currentJob.Id, jobFields)

			if updateErr != nil {
				updateErr = fmt.Errorf("failed to update job status to File Transmission Failed for file %s: %s", currentJob.FullInputFileLocation(), updateErr.Error())
				errs = multierror.Append(errs, updateErr)
			}
			err = fmt.Errorf("failed to send NotifyNewFile request for file %s: %s", currentJob.FullInputFileLocation(), err.Error())
			errs = multierror.Append(errs, err)
			continue
		}
		// send the file to the receiver
		actualRetryAttempts, err := fh.fileReceiverClient.TransmitFile(currentJob.Id, currentJob.InputFile)
		if err != nil {
			jobFields := make(map[string]interface{})
			if !helpers.IsNetworkError(err) {
				jobFields[types.JobStatus] = pkg.StatusTransmissionFailed
			} else {
				jobFields[types.JobStatus] = pkg.StatusIncomplete
			}

			jobFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderOem
			jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileTransmitting.Error()
			_, updateErr := fh.jobRepoClient.Update(currentJob.Id, jobFields)

			if updateErr != nil {
				updateErr = fmt.Errorf("failed to update job status to File Transmission Failed for file %s: %s", currentJob.FullInputFileLocation(), updateErr.Error())
				errs = multierror.Append(errs, updateErr)
			}
			err = fmt.Errorf("failed to Transmit File %s: %s", currentJob.FullInputFileLocation(), err.Error())
			errs = multierror.Append(errs, err)
			continue
		}

		fh.lc.Debug("Number of retries attempted to transmit input file %s: %d ", currentJob.FullInputFileLocation(), actualRetryAttempts)
	}
	return errs
}

// The retry function is a wrapper for the RetryOnStartup call that is utilized by the retry endpoint
func (fh *FileHandler) retry(writer http.ResponseWriter, request *http.Request) {
	err := fh.RetryOnStartup()
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	fh.lc.Debug("Retry endpoint successfully called")
}

// SendNewFile checks to see if the file exists before the file sender takes ownership. Once the job owner is
// updated, the sender will send the job and then the file to the receiver.
func (fh *FileHandler) SendNewFile(writer http.ResponseWriter, request *http.Request) {
	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, requestBody)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer,
			fmt.Errorf("failed to process SendNewFile request:  (%s): %s", request.URL.String(), err.Error()),
			http.StatusBadRequest)
		return
	}

	var fileJob types.Job
	err = json.Unmarshal(requestBody, &fileJob)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer,
			fmt.Errorf("failed to unmarshal NotifyNewFile request: %s", err.Error()),
			http.StatusBadRequest)
		return
	}

	// validate that the message is good/file info is present - what field the sender needs to do its job
	err = fh.validate(fileJob.InputFile)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer,
			fmt.Errorf("failed to validate NotifyNewFile request for file %s: %s", fileJob.FullInputFileLocation(), err.Error()),
			http.StatusBadRequest)
		return
	}

	// set file sender as owner
	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerFileSenderOem
	fileJob, err = fh.jobRepoClient.Update(fileJob.Id, jobFields)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer,
			fmt.Errorf("failed to update owner in Data Repo request for file %s: %s", fileJob.FullInputFileLocation(), err.Error()),
			http.StatusInternalServerError)
		return
	}

	fh.lc.Debugf("Took ownership of Job for %s", fileJob.FullInputFileLocation())

	// ack back to data-organizer
	writer.WriteHeader(http.StatusOK)

	// upload the job
	failedTransmissionFields := make(map[string]interface{})

	err = fh.fileReceiverClient.TransmitJob(fileJob)
	if err != nil {
		fh.lc.Errorf("failed to transmit job for file %s: %s", fileJob.FullInputFileLocation(), err.Error())
		if !helpers.IsNetworkError(err) {
			failedTransmissionFields[types.JobStatus] = pkg.StatusTransmissionFailed
		} else {
			failedTransmissionFields[types.JobStatus] = pkg.StatusIncomplete
		}
		failedTransmissionFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderOem
		failedTransmissionFields[types.JobErrorDetailsErrorMsg] = pkg.ErrTransmitJob.Error()
		_, err := fh.jobRepoClient.Update(fileJob.Id, failedTransmissionFields)
		if err != nil {
			fh.lc.Errorf("failed to update Data Repo Job Transmission Failed for file %s: %s", fileJob.FullInputFileLocation(), err.Error())
		}
		return
	}

	fh.lc.Debugf("Transmitted Job object for %s", fileJob.FullInputFileLocation())

	// send the file to the receiver
	actualRetryAttempts, err := fh.fileReceiverClient.TransmitFile(fileJob.Id, fileJob.InputFile)
	if err != nil {
		if !helpers.IsNetworkError(err) {
			failedTransmissionFields[types.JobStatus] = pkg.StatusTransmissionFailed
		} else {
			failedTransmissionFields[types.JobStatus] = pkg.StatusIncomplete
		}
		failedTransmissionFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderOem
		failedTransmissionFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileTransmitting.Error()
		if _, err := fh.jobRepoClient.Update(fileJob.Id, failedTransmissionFields); err != nil {
			fh.lc.Errorf("failed to update Data Repo File Transmission Failed for file %s: %s", fileJob.FullInputFileLocation(), err.Error())
		}
		fh.lc.Errorf("failed to Transmit File %s: %s", fileJob.FullInputFileLocation(), err.Error())
		return
	}
	fh.lc.Debugf("Number of retries attempted to transmit input file %s: %d ", fileJob.FullInputFileLocation(), actualRetryAttempts)
	fh.lc.Debugf("Transmitted file for %s", fileJob.FullInputFileLocation())
}

func validateFileName(fileName string) error {
	// Check for path traversal characters
	if strings.Contains(fileName, "..") || strings.HasPrefix(fileName, "/") {
		return errors.New("invalid file path")
	}
	return nil
}

func validateDirName(fileName string) error {
	// Check for path traversal characters
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "./") || strings.Contains(fileName, "..\\") {
		return errors.New("invalid file path")
	}
	return nil
}

// validate is a helper function to check that the fileHostname matches in the job and that the file exists on the file
// system.
func (fh *FileHandler) validate(fileInfo types.FileInfo) error {

	if err := validateFileName(fileInfo.Name); err != nil {
		return err
	}

	if err := validateDirName(fileInfo.DirName); err != nil {
		return err
	}

	// check if the fileHostname matches
	if fileInfo.Hostname != fh.fileHostname {
		return fmt.Errorf("fileHostname does not match: got %s, expected %s", fileInfo.Hostname, fh.fileHostname)
	}

	// check that file exists - if not fail.
	fileName, err := filepath.Abs(filepath.Join(fileInfo.DirName, fileInfo.Name))
	if err != nil {
		return fmt.Errorf("error accessing file: %s", err.Error())
	}
	_, err = os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("error accessing file: %s", err.Error())
	}
	return err
}
