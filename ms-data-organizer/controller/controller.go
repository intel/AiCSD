/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/wait"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"aicsd/ms-data-organizer/clients/task_launcher"
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

type DataOrgController struct {
	lc                 logger.LoggingClient
	jobRepoClient      job_repo.Client
	fileSenderClient   job_handler.Client
	taskLauncherClient task_launcher.Client
	DependentServices  wait.Services
	AttributeParser    map[string]types.AttributeInfo
}

func New(lc logger.LoggingClient, jobRepoClient job_repo.Client, fileSenderClient job_handler.Client,
	taskLauncherClient task_launcher.Client, attributeParser map[string]types.AttributeInfo, dependentServices wait.Services) *DataOrgController {
	return &DataOrgController{
		lc:                 lc,
		jobRepoClient:      jobRepoClient,
		fileSenderClient:   fileSenderClient,
		taskLauncherClient: taskLauncherClient,
		DependentServices:  dependentServices,
		AttributeParser:    attributeParser,
	}
}

func (c *DataOrgController) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointNotifyNewFile, c.NotifyNewFileHandler, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointNotifyNewFile)
	}
	err = service.AddRoute(pkg.EndpointRetry, c.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	return nil
}

// RetryOnStartup gets all job entries that the data organizer owns, checks each entry for matching tasks,
// and if there are tasks, sends the entry to the file sender
func (c *DataOrgController) RetryOnStartup() error {
	var err, errs error
	jobs, err := c.jobRepoClient.RetrieveAllByOwner(pkg.OwnerDataOrg)
	if err != nil {
		err = fmt.Errorf("%s %s data: %s", pkg.OwnerDataOrg, pkg.ErrRetrieving, err.Error())
		return err
	}
	for _, currentJob := range jobs {
		c.lc.Debugf("Processing existing Job on restart for %s", currentJob.FullInputFileLocation())
		hasTasks, err := c.taskLauncherClient.MatchTask(currentJob)
		if err != nil {
			err = fmt.Errorf("call failed for matchTasks on file (%s): %s", currentJob.InputFile.Name, err.Error())
			errs = multierror.Append(errs, err)
			continue
		}
		// No tasks - write back to job repo state
		if !hasTasks {
			jobFields := make(map[string]interface{})
			jobFields[types.JobOwner] = pkg.OwnerNone
			jobFields[types.JobStatus] = pkg.StatusNoPipeline
			jobFields[types.JobErrorDetailsOwner] = pkg.OwnerDataOrg
			jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrJobNoMatchingTask.Error()
			_, err = c.jobRepoClient.Update(currentJob.Id, jobFields)
			if err != nil {
				err = fmt.Errorf("%s for job id %s: %s", pkg.ErrUpdating, currentJob.Id, err.Error())
				errs = multierror.Append(errs, err)
				// TODO: add retry logic here?
				continue
			}
			c.lc.Debug("No matching tasks for job id %s", currentJob.Id)
			continue
		}

		c.lc.Debugf("Attempting to pass file %s to sender with err details %s %s", currentJob.FullInputFileLocation(), currentJob.ErrorDetails.Owner, currentJob.ErrorDetails.Error)

		// Send request to file sender
		err = c.fileSenderClient.HandleJob(currentJob)
		if err != nil {
			err = werrors.WrapMsgf(err, "for client: %s", pkg.OwnerFileSenderGateway)
			errs = multierror.Append(errs, err)
			continue
		}

		c.lc.Debugf("File %s passed to sender", currentJob.FullInputFileLocation())

	}
	return errs
}

// The retry function is a wrapper for the RetryOnStartup call, used by the retry endpoint
func (c *DataOrgController) retry(writer http.ResponseWriter, request *http.Request) {
	err := c.RetryOnStartup()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	c.lc.Debug("Retry endpoint successfully called")
}

// NotifyNewFileHandler is a request that gets called to notify the data organizer of a new file that has been written
// to the file system. The new file entry is created in the job repo, checked for tasks on the task launcher, and
// transmitted to the file receiver
func (c *DataOrgController) NotifyNewFileHandler(writer http.ResponseWriter, request *http.Request) {

	var response []byte
	var err error

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)

	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("failed to process NotifyNewFile request:  (%s): %s", request.URL.String(), err.Error()), http.StatusInternalServerError)
		return
	}

	var jobEntry types.Job
	err = json.Unmarshal(requestBody, &jobEntry)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("failed to unmarshal NotifyNewFile request (%s): %s", request.URL.String(), err.Error()), http.StatusInternalServerError)
		return
	}

	err = jobEntry.InputFile.ParseFilenameForAttributes(c.AttributeParser)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
	}

	if len(jobEntry.InputFile.Attributes) <= 0 {
		c.lc.Infof("No attributes found for filename %s", jobEntry.InputFile.Name)
	} else {
		c.lc.Infof("Attributes found for filename %s, %v", jobEntry.InputFile.Name, jobEntry.InputFile.Attributes)
	}

	// 1. set data organizer as owner in the job object and update the status
	jobEntry.Owner = pkg.OwnerDataOrg
	jobEntry.Status = pkg.StatusIncomplete
	// TODO: add any additional information to the job here

	// 2. Create a new job object in the job repository. (Data repository will be responsible for adding the id)
	//    -- response code: 200 if created + id is returned, update local copy
	//    --  				409 if duplicate entry
	// 	  -- 				40X handle other errors
	id, isNew, err := c.jobRepoClient.Create(jobEntry)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("%s for file (%s): %s", pkg.ErrJobCreation, jobEntry.InputFile.Name, err.Error()), http.StatusBadRequest)
		return
	}

	if isNew {
		jobEntry.Id = id
		c.lc.Debugf("Created and took ownership of Job for %s", jobEntry.FullInputFileLocation())
	} else {
		jobEntry, err = c.jobRepoClient.RetrieveById(id)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("%s %s data: %s", pkg.OwnerDataOrg, pkg.ErrRetrieving, err.Error()), http.StatusBadRequest)
			return
		}
		// to cover the case where the file exists and has already been processed
		if jobEntry.Owner != pkg.OwnerDataOrg && jobEntry.Status != pkg.StatusIncomplete {
			writer.WriteHeader(http.StatusAlreadyReported)
			c.lc.Debugf("job for %s has already been processed", jobEntry.FullInputFileLocation())
			return
		}
	}

	// 3. Check to see if there are any tasks that match the current job
	hasTasks, err := c.taskLauncherClient.MatchTask(jobEntry)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("call failed for matchTask on id %s: %s", jobEntry.Id, err.Error()), http.StatusBadRequest)
		return
	}
	// No tasks - write back to job repo state
	if !hasTasks {
		jobFields := make(map[string]interface{})
		jobFields[types.JobOwner] = pkg.OwnerNone
		jobFields[types.JobStatus] = pkg.StatusNoPipeline
		jobFields[types.JobErrorDetailsOwner] = pkg.OwnerDataOrg
		jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrJobNoMatchingTask.Error()
		_, err = c.jobRepoClient.Update(jobEntry.Id, jobFields)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("%s for id %s: %s", pkg.ErrUpdating, jobEntry.Id, err.Error()), http.StatusInternalServerError)
			// TODO: add retry logic here?
			return
		}
		c.lc.Debugf("No matching tasks for Job with %s", jobEntry.FullInputFileLocation())
		http.Error(writer, fmt.Sprintf("No matching tasks for id %s", jobEntry.Id), http.StatusNoContent)
		return
	}
	// 4. If there are tasks, send request to the file sender to send the file (with its full file path) with the given job
	err = c.fileSenderClient.HandleJob(jobEntry)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapMsgf(err, "for client: %s", pkg.OwnerFileSenderGateway), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write(response)

	c.lc.Debugf("Job for %s passed to sender", jobEntry.FullInputFileLocation())
}
