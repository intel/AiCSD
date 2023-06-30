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
	"strings"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"aicsd/as-task-launcher/config"
	"aicsd/as-task-launcher/persist"
	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_handler"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
)

const (
	MqttResultsTopic = "pipeline/inferenceResults"
	CustomTopic      = "pipeline-inference-results"
)

type Controller struct {
	lc                logger.LoggingClient
	persist           persist.Persistence
	jobRepoClient     job_repo.Client
	fileSenderClient  job_handler.Client
	publisher         interfaces.BackgroundPublisher
	service           interfaces.ApplicationService
	config            *config.Configuration
	DependentServices wait.Services
}

func New(lc logger.LoggingClient, persist persist.Persistence, jobRepoClient job_repo.Client,
	fileSenderClient job_handler.Client, publisher interfaces.BackgroundPublisher,
	service interfaces.ApplicationService, config *config.Configuration) *Controller {
	return &Controller{
		lc:                lc,
		persist:           persist,
		jobRepoClient:     jobRepoClient,
		fileSenderClient:  fileSenderClient,
		publisher:         publisher,
		service:           service,
		config:            config,
		DependentServices: wait.Services{wait.ServiceConsul, wait.ServiceRedis, wait.ServiceJobRepo},
	}
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred
func (c *Controller) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointTask, c.Create, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "create")
	}

	err = service.AddRoute(pkg.EndpointTask, c.Get, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "get")
	}

	err = service.AddRoute(pkg.EndpointTaskId, c.GetById, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointTaskId)
	}

	err = service.AddRoute(pkg.EndpointTaskId, c.Delete, http.MethodDelete)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "delete")
	}

	err = service.AddRoute(pkg.EndpointTask, c.Update, http.MethodPut)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "update")
	}

	err = service.AddRoute(pkg.EndpointMatchTask, c.MatchTask, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointMatchTask)
	}

	err = service.AddRoute(pkg.EndpointDataToHandle, c.HandleNewJob, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointDataToHandle)
	}

	err = service.AddRoute(pkg.EndpointPipelineStatus, c.PipelineStatus, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointPipelineStatus)
	}
	err = service.AddRoute(pkg.EndpointRetry, c.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	return nil
}

// Create is a request to post a task in the TaskRepo
func (c *Controller) Create(writer http.ResponseWriter, request *http.Request) {
	task, httpStatus, err := helpers.UnmarshalTask(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	// Check if TaskId already present or not
	if task.Id != "" {
		helpers.HandleErrorMessage(c.lc, writer, taskPkg.ErrDuplicateTaskId, http.StatusBadRequest)
		return
	}

	currentTaskId, err := c.persist.Create(task)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapErr(err, taskPkg.ErrTaskCreation), http.StatusInternalServerError)
		return
	}

	//Successful task creation will result in status set as "StatusCreated"
	writer.WriteHeader(http.StatusCreated)
	writer.Header().Set("Content-Type", "text/plain")
	_, err = writer.Write([]byte(currentTaskId))
	if err != nil {
		c.lc.Errorf("failed to write http response (%s): %s", task.Id, err.Error())
	}

}

// Get is a request to retrieve all tasks from TaskRepo
func (c *Controller) Get(writer http.ResponseWriter, request *http.Request) {
	allTasks, err := c.persist.GetAll()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapErr(err, pkg.ErrCallingGet), http.StatusInternalServerError)
		return
	}

	var rspBody []byte

	if len(allTasks) == 0 {
		writer.WriteHeader(http.StatusNoContent)
	}

	rspBody, err = json.Marshal(allTasks)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, taskPkg.ErrMarshallingTask), http.StatusInternalServerError)
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(rspBody)
	if err != nil {
		c.lc.Errorf("failed to write http response: %s", err.Error())
	}

}

// GetById is a request to retrieve matching task for a specified id.
func (c *Controller) GetById(writer http.ResponseWriter, request *http.Request) {
	id, err := helpers.GetByKeyFromRequest(request, pkg.TaskIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	task, err := c.persist.GetById(id)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, taskPkg.ErrTaskRetrieving), http.StatusInternalServerError)
		return
	}

	jsonRsp, err := json.Marshal(task)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, taskPkg.ErrMarshallingTask), http.StatusInternalServerError)
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

func (c *Controller) Update(writer http.ResponseWriter, request *http.Request) {
	task, httpStatus, err := helpers.UnmarshalTask(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	err = c.persist.Update(task)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("failed to update task for Id (%s): %s",
			task.Id, err.Error()), http.StatusNotFound)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (c *Controller) Delete(writer http.ResponseWriter, request *http.Request) {
	id, err := helpers.GetByKeyFromRequest(request, pkg.TaskIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	err = c.persist.Delete(id)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("failed to delete task for Id (%s): %s", id, err.Error()), http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// RetryOnStartup checks all jobs owned by the TaskLauncher, tries to process them
// and returns all errors encountered
func (c *Controller) RetryOnStartup(retryTimeout int64) error {
	var err, errs error
	now := time.Now().UTC().UnixNano()
	// TODO: add retry logic around the Retrieve call in case job repo is not up yet.
	jobs, err := c.jobRepoClient.RetrieveAllByOwner(pkg.OwnerTaskLauncher)
	if err != nil {
		err = fmt.Errorf("could not retrieve %s data: %s", pkg.OwnerTaskLauncher, err.Error())
		return err
	}
	for _, job := range jobs {
		if job.PipelineDetails.Status == pkg.TaskStatusComplete || job.PipelineDetails.Status == pkg.TaskStatusFailed {
			c.lc.Debugf("Task status is complete or failed for Job with input file %s", job.FullInputFileLocation())
			// update job object with its status if all tasks complete and no output file
			if len(job.PipelineDetails.OutputFiles) == 0 {
				jobFields := make(map[string]interface{})
				jobFields[types.JobOwner] = pkg.OwnerNone
				if job.PipelineDetails.Status == pkg.TaskStatusComplete {
					jobFields[types.JobStatus] = pkg.StatusComplete
				} else {
					jobFields[types.JobStatus] = pkg.StatusPipelineError
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrPipelineFailed.Error()
				}
				job, err = c.jobRepoClient.Update(job.Id, jobFields)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("could not update job repo to status to %s for job with input file %s: %s", job.Status, job.FullInputFileLocation(), err.Error()))
					//TODO: add retry logic here
					continue
				}
			} else {
				err = c.passJobToSender(job)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
				continue
			}
		} else if job.PipelineDetails.Status == pkg.TaskStatusProcessing && job.LastUpdated+retryTimeout < now {
			c.lc.Debugf("Resending job to its pipeline for Job with input file %s", job.FullInputFileLocation())
			matchedTask, err := c.matchJobToTasks(job)
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			if matchedTask == nil {
				jobFields := make(map[string]interface{})
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusNoPipeline
				jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
				jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrJobNoMatchingTask.Error()
				_, err = c.jobRepoClient.Update(job.Id, jobFields)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("could not update job repo to status no pipeline for job with input file %s: %s", job.FullInputFileLocation(), err.Error()))
					continue
				}
				// no tasks matched and successfully updated repo, so log warning and return ok
				c.lc.Warnf("no pipeline id matched for job id: %s", job.Id)
				continue
			}
			jobFields := make(map[string]interface{})
			jobFields[types.JobOwner] = pkg.OwnerTaskLauncher
			jobFields[types.JobPipelineTaskId] = matchedTask.Id
			jobFields[types.JobPipelineOutputHost] = c.config.FileHostname
			job, err = c.jobRepoClient.Update(job.Id, jobFields)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("could not update job repo to status no pipeline for job with input file %s: %s", job.FullInputFileLocation(), err.Error()))
				continue
			}
			err = helpers.PublishEventForPipeline(c.publisher, c.service, c.lc, job, matchedTask, c.config.JobRepoBaseUrl, c.config.PipelineStatusBaseUrl, c.config.DeviceProfileName, c.config.DeviceName, c.config.ResourceName)
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
		} else {
			c.lc.Debugf("No action to retry for Job with input file %s", job.FullInputFileLocation())
		}
	}
	return errs
}

// This struct is utilized in retry endpoint call
type TimeDuration struct {
	TimeoutDuration string `json:"TimeoutDuration"`
}

// The retry function is a wrapper for the RetryOnStartup call, used by the retry endpoint
func (c *Controller) retry(writer http.ResponseWriter, request *http.Request) {
	var err error
	requestBody := make([]byte, request.ContentLength)

	_, err = io.ReadFull(request.Body, requestBody)
	if err != nil {
		werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String())
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	// capture JSON result as defined struct for ease of use
	var result TimeDuration
	err = json.Unmarshal(requestBody, &result)
	if err != nil {
		err = werrors.WrapMsgf(err, "failed to unmarshal request (%s)", request.URL.String())
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	timeoutDur, err := time.ParseDuration(result.TimeoutDuration)
	if err != nil {
		err = werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String())
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	err = c.RetryOnStartup(timeoutDur.Nanoseconds())
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	c.lc.Debug("Retry endpoint successfully called")
}

// MatchTask checks the job to see if there are any tasks that match and returns true or false based on whether
// there are matches or not
func (c *Controller) MatchTask(writer http.ResponseWriter, request *http.Request) {
	job, httpStatus, err := helpers.UnmarshalJob(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	matchedTask, err := c.matchJobToTasks(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/plain")
	// check pipeline id to see if the task is empty
	if matchedTask == nil {
		writer.Write([]byte("false"))
		return
	}
	writer.Write([]byte("true"))
}

// HandleNewJob takes a job and matches it to the tasks that need to run,
// sets the task launcher as the owner, adds the task information, and publishes the event for the task
func (c *Controller) HandleNewJob(writer http.ResponseWriter, request *http.Request) {
	job, httpStatus, err := helpers.UnmarshalJob(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	// call match task and set info to metadata object
	matchedTask, err := c.matchJobToTasks(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	if matchedTask == nil {
		jobFields := make(map[string]interface{})
		jobFields[types.JobOwner] = pkg.OwnerNone
		jobFields[types.JobStatus] = pkg.StatusNoPipeline
		jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
		jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrJobNoMatchingTask.Error()
		_, err = c.jobRepoClient.Update(job.Id, jobFields)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("could not update job repo to status no pipeline for job id %s: %s", job.Id, err.Error()), http.StatusInternalServerError)
			//TODO: add retry logic here
			return
		}
		// no tasks matched and successfully updated repo, so log warning and return ok
		c.lc.Warnf("no pipeline id matched for job id: %s", job.Id)
		return
	}

	// take ownership, add task info to pipeline details, update the data repo and ack back
	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerTaskLauncher
	jobFields[types.JobPipelineTaskId] = matchedTask.Id
	jobFields[types.JobPipelineOutputHost] = c.config.FileHostname
	jobFields[types.JobPipelineStatus] = pkg.TaskStatusProcessing
	job, err = c.jobRepoClient.Update(job.Id, jobFields)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("could not update job repo for job id %s: %s", job.Id, err.Error()), http.StatusInternalServerError)
		//TODO: add retry logic here
		return
	}

	c.lc.Debugf("Took ownership of Job for %s", job.FullInputFileLocation())

	writer.WriteHeader(http.StatusOK)

	err = helpers.PublishEventForPipeline(c.publisher, c.service, c.lc, job, matchedTask, c.config.JobRepoBaseUrl, c.config.PipelineStatusBaseUrl, c.config.DeviceProfileName, c.config.DeviceName, c.config.ResourceName)
	if err != nil {
		c.lc.Error(err.Error())
	}

}

// PipelineStatus takes the task status and handles the job after the pipeline has been completed
// It will send a file if needed
func (c *Controller) PipelineStatus(writer http.ResponseWriter, request *http.Request) {
	// unpack the job id, task id, task status
	jobId, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	taskId, err := helpers.GetByKeyFromRequest(request, pkg.TaskIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	taskStatus, err := c.getStatusFromRequestBody(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	if taskStatus != pkg.TaskStatusFailed && taskStatus != pkg.TaskStatusComplete {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("task status not set to failed or complete for job id %s, taskid %s", jobId, taskId), http.StatusBadRequest)
		return
	}
	// load fresh copy of job by id
	job, err := c.jobRepoClient.RetrieveById(jobId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	if job.PipelineDetails.TaskId != taskId {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("task id mismatch: got %s, expected %s", taskId, job.PipelineDetails.TaskId),
			http.StatusBadRequest)
		return
	}
	// verify that the current pipeline status matches what is passed in
	if job.PipelineDetails.Status != taskStatus {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("task status mismatch for job id %s, got %s, expected %s", jobId, taskStatus, job.PipelineDetails.Status),
			http.StatusBadRequest)
		return
	}

	c.lc.Debugf("Received pipeline status of '%s' for Job with input file %s", taskStatus, job.FullInputFileLocation())

	// update job object with its status if all tasks complete
	if len(job.PipelineDetails.OutputFiles) == 0 {
		jobFields := make(map[string]interface{})
		jobFields[types.JobOwner] = pkg.OwnerNone
		if job.PipelineDetails.Status == pkg.TaskStatusComplete {
			jobFields[types.JobStatus] = pkg.StatusComplete
		} else {
			jobFields[types.JobStatus] = pkg.StatusPipelineError
			jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
			jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrPipelineFailed.Error()
		}
		_, err = c.jobRepoClient.Update(job.Id, jobFields)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("could not update job repo to status to %s for job id %s: %s", jobFields[types.JobStatus], job.Id, err.Error()), http.StatusInternalServerError)
			//TODO: add retry logic here
			return
		}

		c.lc.Debugf("No resulting output file from pipeline for Job with %s, set owner to %s and status to %s", job.FullInputFileLocation(), pkg.OwnerNone, job.Status)
	} else {
		err = c.passJobToSender(job)
		if err != nil {
			c.lc.Error(err.Error())
		}

		c.lc.Debugf("Passed Job for output file(s) %s to sender", job.FullOutputFileLocation())
	}

	writer.WriteHeader(http.StatusOK)

	if job.PipelineDetails.Results != "" {
		err := c.publishResultsForJob(job)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("error publishing results for job"),
				http.StatusBadRequest)
			return
		}
	}
}

// getStatusFromRequestBody is used to read the request body and match it to a status
func (c *Controller) getStatusFromRequestBody(request *http.Request) (string, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return "", werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String())
	}
	status := strings.TrimSpace(string(body))
	if strings.EqualFold(status, pkg.TaskStatusFailed) {
		return pkg.TaskStatusFailed, nil
	}
	if strings.EqualFold(status, pkg.TaskStatusComplete) {
		return pkg.TaskStatusComplete, nil
	}
	if strings.EqualFold(status, pkg.TaskStatusProcessing) {
		return pkg.TaskStatusProcessing, nil
	}
	if strings.EqualFold(status, pkg.TaskStatusFileNotFound) {
		return pkg.TaskStatusFileNotFound, nil
	}
	return "", taskPkg.ErrReqBodyTaskStatusMismatch
}

// matchJobToTasks will match a Job to its corresponding Task and return the first task that matches
func (c *Controller) matchJobToTasks(job types.Job) (*types.Task, error) {
	c.lc.Debugf("Matching to Tasks for Job with input file %s", job.FullInputFileLocation())
	tasks, err := c.persist.GetAll()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve tasks: %s", err.Error())
	}

	// iterate over tasks
	for _, task := range tasks {
		isMatch, err := helpers.ApplyJsonLogicToJob(job, task.JobSelector)
		if err != nil {
			return nil, werrors.WrapMsgf(err, "could not apply json logic for job id: %s, task id %s", job.Id, task.Id)
		}
		if isMatch {
			return &task, nil
		}
	}
	return nil, nil
}

// passJobToSender will take the job object, validate its output file location and determine if handle job should be called
func (c *Controller) passJobToSender(job types.Job) error {
	c.lc.Debugf("Passing output file to sender for Job with input file %s", job.FullInputFileLocation())

	// init output file err and status fields with owner of file sender gw as publishing job to message bus
	for fileId, file := range job.PipelineDetails.OutputFiles {
		job.UpdateOutputFile(fileId, file.DirName, file.Name, file.Extension, "", "", nil, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway)
	}

	err := job.ValidateHost(c.config.FileHostname)
	if err != nil {
		jobErr := pkg.CreateUserFacingError(pkg.OwnerTaskLauncher, pkg.ErrJobInvalid)
		_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusPipelineError, pkg.TaskStatusFileNotFound, jobErr, "", "", nil)
		if updateErr != nil {
			return fmt.Errorf("could not update job repo task status for invalid job hostname for job id %s: %s", job.Id, updateErr.Error())
		}
		c.lc.Warnf("invalid hostname for job id %s: %s", job.Id, err.Error())
	}

	err = job.ValidateFiles()
	if err != nil {
		jobErr := pkg.CreateUserFacingError(pkg.OwnerTaskLauncher, pkg.ErrFileInvalid)
		_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusPipelineError, pkg.TaskStatusFileNotFound, jobErr, "", "", job.PipelineDetails.OutputFiles)
		if updateErr != nil {
			return fmt.Errorf("could not update job repo task status to file not found for job id %s: %s", job.Id, updateErr.Error())
			//TODO: add retry logic here
		}
		c.lc.Warnf("files %s not found for job id %s: %v", job.PipelineDetails.OutputFiles, job.Id, err)
		return nil
	}

	err = c.fileSenderClient.HandleJob(job)
	if err != nil {
		return fmt.Errorf("handle job call failed: %v", err)
	}
	return nil
}

// publishResultsForJob publishes the pipeline results to the EdgeX Message Bus using to the topic InferenceResults
func (c *Controller) publishResultsForJob(job types.Job) error {
	// publish event
	ctx := c.service.BuildContext(uuid.NewString(), common.ContentTypeText)
	ctx.AddValue(pkg.PublishTopicKey, MqttResultsTopic)
	ctx.AddValue(pkg.CustomTopicKey, CustomTopic)
	msg := fmt.Sprintf("filename:%s; results:%s", strings.ReplaceAll(job.FullInputFileLocation(), ":", "="), job.PipelineDetails.Results)
	err := c.publisher.Publish([]byte(msg), ctx)
	if err != nil {
		return fmt.Errorf("could not publish results for job id %s, job input file %s: %s", job.Id, job.FullInputFileLocation(), err.Error())
	}
	c.lc.Debugf("Publish Results for job id=%s, input file=%s", job.Id, job.FullInputFileLocation())
	return nil
}
