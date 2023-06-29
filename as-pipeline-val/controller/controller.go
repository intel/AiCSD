/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/as-pipeline-val/clients"
	"aicsd/as-pipeline-val/config"
	simTypes "aicsd/as-pipeline-val/types"
	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/google/uuid"

	"aicsd/pkg"
	"aicsd/pkg/werrors"
)

type Controller struct {
	lc                logger.LoggingClient
	DependentServices wait.Services
	publisher         interfaces.BackgroundPublisher
	service           interfaces.ApplicationService
	config            *config.Configuration
	pipelineClient    clients.Client
	jobs              []types.Job
}

func New(lc logger.LoggingClient, publisher interfaces.BackgroundPublisher, service interfaces.ApplicationService, config *config.Configuration, pipelineClient clients.Client) *Controller {
	return &Controller{
		lc:                lc,
		DependentServices: wait.Services{wait.ServiceConsul, wait.ServiceRedis},
		publisher:         publisher,
		service:           service,
		config:            config,
		pipelineClient:    pipelineClient,
		jobs:              []types.Job{},
	}
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred.
func (c *Controller) RegisterRoutes(service interfaces.ApplicationService) error {
	if err := service.AddRoute(pkg.EndpointLaunchPipeline, c.LaunchEventForPipeline, http.MethodPost); err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointLaunchPipeline)
	}

	if err := service.AddRoute(pkg.EndpointGetPipelines, c.GetPipelines, http.MethodGet); err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointLaunchPipeline)
	}
	err := service.AddRoute(pkg.EndpointJobPipeline, c.UpdateJobPipeline, http.MethodPut)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "UpdateJobPipeline")
	}

	err = service.AddRoute(pkg.EndpointPipelineStatus, c.PipelineStatus, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointPipelineStatus)
	}

	err = service.AddRoute(pkg.EndpointJob, c.GetJobs, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "GetAll")
	}

	c.lc.Info("Routes added...")
	return nil
}

// LaunchEventForPipeline is an endpoint that will create a job and publish it on the specified mqtt topic.
// It returns: - 201 if the job was created and the event was sent.
//   - 400 if the request payload is incorrect.
//   - 500 if the job could not be published.
func (c *Controller) LaunchEventForPipeline(writer http.ResponseWriter, request *http.Request) {

	// grab data from payload
	requestBody := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, requestBody)
	if err != nil {
		err = werrors.WrapMsgf(err, "unable to read request body")
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	var inputInfo simTypes.LaunchInfo
	err = json.Unmarshal(requestBody, &inputInfo)
	if err != nil {
		err = werrors.WrapMsgf(err, "failed to unmarshal request")
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	// create job
	job := c.createJob(inputInfo)
	simTask := types.Task{
		Id:               job.PipelineDetails.TaskId,
		Description:      "sim simTask",
		JobSelector:      "",
		PipelineId:       inputInfo.PipelineTopic,
		ResultFileFolder: inputInfo.OutputFileFolder,
		ModelParameters:  map[string]string{},
	}
	simTask.SetLastUpdated()
	// publish job to topic
	err = helpers.PublishEventForPipeline(c.publisher, c.service, c.lc, job, &simTask, c.config.SimBaseUrl, c.config.SimBaseUrl, c.config.DeviceProfileName, c.config.DeviceName, c.config.ResourceName)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusCreated)
}

// UpdateJobPipeline is an endpoint that will update the job entry.
// It returns: - 200 if the job was updated successfully.
//   - 400 if the request payload is incorrect.
//   - 500 if the job could not be processed.
func (c *Controller) UpdateJobPipeline(writer http.ResponseWriter, request *http.Request) {
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
	jobIndex, err := strconv.Atoi(jobId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	job := c.jobs[jobIndex]

	if taskId != job.PipelineDetails.TaskId {
		helpers.HandleErrorMessage(c.lc, writer, errors.New("url taskid parameter does not match taskid for specified job"),
			http.StatusBadRequest)
		return
	}

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf(pkg.ErrFmtProcessingReq, request.URL.String()), http.StatusInternalServerError)
		return
	}

	pipelineDetails := types.PipelineInfo{}

	err = json.Unmarshal(requestBody, &pipelineDetails)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("failed to unmarshal request (%s): %s",
			request.URL.String(), err.Error()), http.StatusInternalServerError)
		return
	}

	// Validate the passed in pipeline details. Status is the only required value
	if pipelineDetails.Status != pkg.TaskStatusComplete &&
		pipelineDetails.Status != pkg.TaskStatusFailed {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("invalid Status value '%s'", pipelineDetails.Status),
			http.StatusBadRequest)
		return
	}

	c.jobs[jobIndex].PipelineDetails.Status = pipelineDetails.Status
	c.jobs[jobIndex].PipelineDetails.QCFlags = pipelineDetails.QCFlags
	c.jobs[jobIndex].PipelineDetails.OutputFiles = pipelineDetails.OutputFiles
	c.jobs[jobIndex].PipelineDetails.Results = pipelineDetails.Results
}

// PipelineStatus is an endpoint that ensures all tasks are complete for the job so that processing may continue.
// It returns: - 201 if the job was created and the event was sent.
//   - 400 if the request payload is incorrect.
//   - 500 if the job could not be published.
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
	jobIndex, err := strconv.Atoi(jobId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	job := c.jobs[jobIndex]
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
		c.jobs[jobIndex].Owner = pkg.OwnerNone
		if job.PipelineDetails.Status == pkg.TaskStatusComplete {
			c.jobs[jobIndex].Status = pkg.StatusComplete
		} else {
			c.jobs[jobIndex].Status = pkg.StatusPipelineError
			c.jobs[jobIndex].ErrorDetails = pkg.CreateUserFacingError(simTypes.OwnerPipelineValidator, pkg.ErrPipelineFailed)
		}

		c.lc.Debugf("No resulting output file from pipeline for Job with %s, set owner to %s and status to %s", job.FullInputFileLocation(), pkg.OwnerNone, job.Status)
	}
}

// GetPipelines is an endpoint that will query the pipeline service for the current pipelines.
// It returns: - 200 if the pipelines are fetched and returns the fetched pipelines.
//   - 204 if no pipelines were returned
//   - 400 if the request cannot be made.
//   - 500 if the pipeline information cannot be marshaled.
func (c *Controller) GetPipelines(writer http.ResponseWriter, _ *http.Request) {

	res, err := c.pipelineClient.GetPipelines()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrCallingGet), http.StatusBadRequest)
		return
	}

	if len(res) == 0 {
		helpers.HandleErrorMessage(c.lc, writer,
			errors.New("no pipelines retrieved"), http.StatusNoContent)
	}

	jsonRsp, err := json.Marshal(res)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

// GetJobs retrieves all the current in memory jobs and returns them.
// It returns: - 200 if the pipelines are fetched and returns the fetched pipelines.
//   - 500 if the job could not be published.
func (c *Controller) GetJobs(writer http.ResponseWriter, _ *http.Request) {
	jsonRsp, err := json.Marshal(c.jobs)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

func (c *Controller) createJob(inputInfo simTypes.LaunchInfo) types.Job {
	filepath, filename := path.Split(inputInfo.InputFileLocation)
	job := types.Job{
		Id:    fmt.Sprint(len(c.jobs)),
		Owner: simTypes.OwnerPipelineValidator,
		InputFile: types.FileInfo{
			Hostname:   "gateway",
			DirName:    filepath,
			Name:       filename,
			Extension:  path.Ext(filename),
			Attributes: map[string]string{},
		},
		PipelineDetails: types.PipelineInfo{
			TaskId: uuid.NewString(),
			Status: pkg.TaskStatusProcessing,
		},
		Status: pkg.StatusIncomplete,
	}
	job.SetLastUpdated()
	c.jobs = append(c.jobs, job)
	return job
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
