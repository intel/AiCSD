/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/wait"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"aicsd/ms-file-receiver-gateway/config"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_handler"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/hashicorp/go-multierror"
)

const windowsFilepathPrefix = ":\\"

type FileHandler struct {
	lc                 logger.LoggingClient
	jobRepoClient      job_repo.Client
	taskLauncherClient job_handler.Client
	jobMap             map[string]types.Job
	baseFileFolder     string
	fileHostname       string
	DependentServices  wait.Services
}

// New is used like a constructor for a file handler client
func New(lc logger.LoggingClient, jobRepoClient job_repo.Client, taskLauncherClient job_handler.Client, config *config.Configuration) *FileHandler {
	return &FileHandler{
		lc:                 lc,
		jobRepoClient:      jobRepoClient,
		taskLauncherClient: taskLauncherClient,
		jobMap:             make(map[string]types.Job),
		baseFileFolder:     config.BaseFileFolder,
		fileHostname:       config.FileHostname,
		DependentServices:  wait.Services{wait.ServiceConsul, wait.ServiceJobRepo, wait.ServiceTaskLauncher},
	}
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred
func (fh *FileHandler) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointTransmitJob, fh.TransmitJob, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointTransmitJob)
	}
	err = service.AddRoute(pkg.EndpointTransmitFile, fh.TransmitFile, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointTransmitFile)
	}
	err = service.AddRoute(pkg.EndpointRetry, fh.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	return nil
}

// RetryOnStartup will be called on startup to look at what job objects the receiver owns and attempts to process
// them. The function checks that the job host name matches and that the file exists before sending to the task
// launcher using the data to handle API.
func (fh *FileHandler) RetryOnStartup() error {
	var err, errs error
	jobs, err := fh.jobRepoClient.RetrieveAllByOwner(pkg.OwnerFileRecvGateway)
	if err != nil {
		return fmt.Errorf("could not retrieve %s data: %s", pkg.OwnerFileRecvGateway, err.Error())
	}
	for _, currentJob := range jobs {
		fh.lc.Debugf("retrying Job with input file %s", currentJob.FullInputFileLocation())
		if currentJob.InputFile.Hostname != fh.fileHostname {
			err = fmt.Errorf("fileHostname does not match: got %s, expected %s", currentJob.InputFile.Hostname, fh.fileHostname)
			errs = multierror.Append(errs, err)
			// TODO: send error back to the job repo?
			continue
		}
		currentFile := filepath.Join(currentJob.InputFile.DirName, currentJob.InputFile.Name)
		_, err = os.Stat(currentFile)
		if err != nil {
			err = fmt.Errorf("file %s not found: %s", currentFile, err.Error())
			errs = multierror.Append(errs, err)
			// TODO: send error back to the job repo?
			continue
		}
		// TODO: when we have checksum, check the checksum of the file
		// call data to handle
		err = fh.taskLauncherClient.HandleJob(currentJob)
		if err != nil {
			err = fmt.Errorf("HandleJob call failed: %s", err.Error())
			errs = multierror.Append(errs, err)
			// TODO: add retry logic here? should this call back to the job repo?
		}

		fh.lc.Debugf("Sent Job for %s to task launcher", currentJob.FullInputFileLocation())
	}

	return errs
}

// The retry function is a wrapper for the RetryOnStartup call, used by the retry endpoint
func (fh *FileHandler) retry(writer http.ResponseWriter, request *http.Request) {
	err := fh.RetryOnStartup()
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	fh.lc.Debug("Retry endpoint successfully called")
}

// TransmitJob is used to process a post request that contains the job object. The job object is  saved
// to the local app memory in a map where the keys are ids and the values are the corresponding job objects.
func (fh *FileHandler) TransmitJob(writer http.ResponseWriter, request *http.Request) {

	var err error

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("failed to process TransmitJob request:  (%s): %s", request.URL.String(), err.Error()), http.StatusBadRequest)
		return
	}

	var fileJob types.Job
	err = json.Unmarshal(requestBody, &fileJob)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("failed to unmarshal TransmitJob request:  (%s): %s", request.URL.String(), err.Error()), http.StatusBadRequest)
		return
	}

	if len(fileJob.Id) == 0 {
		helpers.HandleErrorMessage(fh.lc, writer, errors.New("failed to process TransmitJob request: Id is empty"), http.StatusBadRequest)
		return
	}

	fh.jobMap[fileJob.Id] = fileJob

	// ack back to file-sender
	writer.WriteHeader(http.StatusOK)

	fh.lc.Debugf("Received and cached Job object for %s", fileJob.FullInputFileLocation())
}

// TransmitFile is used to process a post request to transmit a file as the request body. The file is saved to the local
// file system and the job object is updated accordingly.
func (fh *FileHandler) TransmitFile(writer http.ResponseWriter, request *http.Request) {
	// For a multiform approach, use this ref: https://ayada.dev/posts/multipart-requests-in-go/
	// read the request header
	requestFilename := request.Header.Get(pkg.FilenameKey)
	requestId := request.Header.Get(pkg.JobIdKey)

	// Sanitize and validate filename
	if strings.Contains(requestFilename, "/") || strings.Contains(requestFilename, "\\") || strings.Contains(requestFilename, "..") {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("invalid filename: %s", requestFilename), http.StatusBadRequest)
		return
	}

	jobEntry, ok := fh.jobMap[requestId]
	if !ok {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("did not receive job mapping to id (%s):", requestId), http.StatusInternalServerError)
		return
	}

	if jobEntry.InputFile.Name != requestFilename {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("received job input file does not match requested filename (%s):", requestFilename), http.StatusInternalServerError)
		return
	}

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, requestBody)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("failed to process TransmitFile request: %s", err.Error()), http.StatusBadRequest)
		return
	}

	subFolderLinux := ""
	// if-check for windows file-system vs linux file-system
	if strings.Contains(jobEntry.InputFile.DirName, windowsFilepathPrefix) {
		// parse Windows filepath for subfolder
		inputFileDirWinSections := strings.SplitAfter(jobEntry.InputFile.DirName, "\\oem-files\\")
		if len(inputFileDirWinSections) >= 2 {
			subFolderLinux = strings.ReplaceAll(inputFileDirWinSections[1], "\\", "/")
		} else {
			helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("expected valid Windows filepath keyword \"\\oem-files\\\", got (%s)", inputFileDirWinSections[0]), http.StatusInternalServerError)
			return
		}
	} else {
		// parse Linux filepath for subfolder (used in single-device testing)
		subFolderPath := strings.SplitAfter(jobEntry.InputFile.DirName, "/input/")
		subFolderLinux = subFolderPath[len(subFolderPath)-1]
	}
	// inputFileDir is the directory including the subfolders for the input file
	inputFileDir := filepath.Join(fh.baseFileFolder, subFolderLinux)
	// fileLocation is the filepath for the input file
	fileLocation := filepath.Join(inputFileDir, filepath.Base(requestFilename))
	err = os.MkdirAll(inputFileDir, 0777)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("failed to write required folder structure :  (%s): %s", inputFileDir, err.Error()), http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(fileLocation, requestBody, pkg.FilePermissions)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("failed to write file from request:  (%s): %s", fileLocation, err.Error()), http.StatusInternalServerError)
		return
	}

	// update ownership & file location in jobEntry
	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerFileRecvGateway
	jobFields[types.JobInputFileDir] = inputFileDir
	jobFields[types.JobInputFileHost] = fh.fileHostname
	jobFields[types.JobStatus] = pkg.StatusIncomplete

	fh.lc.Debugf("Received and wrote file %s successful at: %s", jobEntry.FullInputFileLocation(), fileLocation)

	// update job repo
	jobEntry, err = fh.jobRepoClient.Update(jobEntry.Id, jobFields)
	if err != nil {
		helpers.HandleErrorMessage(fh.lc, writer, fmt.Errorf("job repo update failed: %s", err.Error()), http.StatusInternalServerError)
		// TODO: clean up the file or retry
		return
	}

	fh.lc.Debugf("Took ownership of Job for %s", jobEntry.FullInputFileLocation())

	// ack back to file-sender
	writer.WriteHeader(http.StatusOK)

	// remove Job Id from jobMap (indicating no further processing on the ID)
	// TODO: only delete Job Id from map and update task launcher once ALL files are transferred
	delete(fh.jobMap, requestId)
	// and send notification to task launcher HandleJob API
	err = fh.taskLauncherClient.HandleJob(jobEntry)
	if err != nil {
		fh.lc.Errorf("task launcher dataToHandle call failed: %s", err.Error())
		return
	}

	fh.lc.Debugf("Passed Job for %s to task launcher", jobEntry.FullInputFileLocation())
}
