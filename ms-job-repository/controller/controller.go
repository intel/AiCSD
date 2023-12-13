/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"aicsd/ms-job-repository/persist"
	"aicsd/pkg"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"
	"aicsd/pkg/werrors"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
)

type JobRepoController struct {
	lc                logger.LoggingClient
	persist           persist.Persistence
	bundle            *i18n.Bundle
	DependentServices wait.Services
}

// New defines a client for the JobRepoController
func New(lc logger.LoggingClient, persist persist.Persistence, bundle *i18n.Bundle) *JobRepoController {
	return &JobRepoController{
		lc:                lc,
		persist:           persist,
		bundle:            bundle,
		DependentServices: wait.Services{wait.ServiceConsul, wait.ServiceRedis},
	}
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred
func (c *JobRepoController) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointJob, c.Create, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "Create")
	}

	err = service.AddRoute(pkg.EndpointJobId, c.Delete, http.MethodDelete)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "Delete")
	}

	err = service.AddRoute(pkg.EndpointJobId, c.Update, http.MethodPut)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "Update")
	}

	err = service.AddRoute(pkg.EndpointJobPipeline, c.UpdatePipeline, http.MethodPut)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "UpdatePipeline")
	}

	err = service.AddRoute(pkg.EndpointJob, c.GetAll, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "GetAll")
	}

	err = service.AddRoute(pkg.EndpointJobId, c.GetById, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "GetById")
	}

	err = service.AddRoute(pkg.EndpointJobOwner, c.GetByOwner, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, "GetByOwner")
	}

	return nil
}

// Create is a request to post a job in the JobRepo.
func (c *JobRepoController) Create(writer http.ResponseWriter, request *http.Request) {
	job, httpStatus, err := helpers.UnmarshalJob(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	status, localJob, err := c.persist.Create(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapErr(err, pkg.ErrJobCreation), http.StatusBadRequest)
		return
	}

	switch status {
	case persist.StatusCreated:
		writer.WriteHeader(http.StatusCreated)
		c.lc.Debugf("Created new Job entry for %s, Id=%s ", job.FullInputFileLocation(), localJob.Id)
		break
	case persist.StatusExists:
		c.lc.Debugf("Job for file %s already exists, returning Id=%s", job.FullInputFileLocation(), localJob.Id)
		writer.WriteHeader(http.StatusConflict)
		break
	}
	writer.Header().Set("Content-Type", "text/plain")
	_, err = writer.Write([]byte(localJob.Id))
	if err != nil {
		c.lc.Errorf(werrors.WrapMsgf(err, pkg.ErrFmtWritingHttpResp, "job id "+job.Id).Error())
	}
}

// GetAll is a request to retrieve all jobs
func (c *JobRepoController) GetAll(writer http.ResponseWriter, request *http.Request) {
	accept := request.Header.Get("Accept-Language")
	c.lc.Debugf("accept language from headers= %s", accept)

	job, err := c.persist.GetAll()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrRetrieving), http.StatusInternalServerError)
		return
	}

	// localize job to proper language
	for k := range job {
		err := job[k].Translate(c.bundle, accept)
		if err != nil {
			// Note: Soft error out on translation errors by logging issue with relevant job information.
			// Fields that did not properly translate will contain the ErrorDetail.Error of pkg.ErrTranslating translated to the correct language.
			c.lc.Errorf("job.Translate() failed with error %v for job %v", pkg.ErrTranslating, job[k])
		}
		c.lc.Debugf("here are my jobs translated %v", job[k])
	}

	jsonRsp, err := json.Marshal(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

// GetById is a request to retrieve matching job for a specified id.
func (c *JobRepoController) GetById(writer http.ResponseWriter, request *http.Request) {
	id, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	job, err := c.persist.GetById(id)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrRetrieving), http.StatusInternalServerError)
		return
	}
	jsonRsp, err := json.Marshal(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

// GetByOwner is a request to retrieve matching jobs for a specified owner.
func (c *JobRepoController) GetByOwner(writer http.ResponseWriter, request *http.Request) {
	owner, err := helpers.GetByKeyFromRequest(request, pkg.OwnerKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	job, err := c.persist.GetByOwner(owner)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrRetrieving), http.StatusInternalServerError)
		return
	}
	jsonRsp, err := json.Marshal(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

// Update is a whole update of the job object
func (c *JobRepoController) Update(writer http.ResponseWriter, request *http.Request) {
	id, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	var jobFields map[string]interface{}

	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String()), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(requestBody, &jobFields)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapErr(err, pkg.ErrUnmarshallingJob), http.StatusBadRequest)
		return
	}

	job, err := c.persist.Update(id, jobFields)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to update job for id (%s)",
			id), http.StatusNotFound)
		return
	}

	jsonRsp, err := json.Marshal(job)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			werrors.WrapErr(err, pkg.ErrMarshallingJob), http.StatusInternalServerError)
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonRsp)
	if err != nil {
		c.lc.Errorf(werrors.WrapErr(err, pkg.ErrWritingHttpResp).Error())
	}
}

// UpdatePipeline updates the job object's pipeline details
func (c *JobRepoController) UpdatePipeline(writer http.ResponseWriter, request *http.Request) {
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

	job, err := c.persist.GetById(jobId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, errors.New("url jobid parameter does not match a job"),
			http.StatusBadRequest)
		return
	}

	if job.PipelineDetails.TaskId != taskId {
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

	// TaskId is passed in URL and is not expected to be in the JSON body, so use the one from the URL.
	// OutputFileHost is previously set and is not expected to be in the JSON body, so use previous value
	jobFields := make(map[string]interface{})
	jobFields[types.JobPipelineStatus] = pipelineDetails.Status
	jobFields[types.JobPipelineQCFlags] = pipelineDetails.QCFlags
	jobFields[types.JobPipelineOutputFiles] = pipelineDetails.OutputFiles
	jobFields[types.JobPipelineResults] = pipelineDetails.Results

	_, err = c.persist.Update(jobId, jobFields)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to update job pipeline details for job id (%s)",
			jobId), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// Delete is a request to remove a job for a specified id
func (c *JobRepoController) Delete(writer http.ResponseWriter, request *http.Request) {
	id, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	err = c.persist.Delete(id)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, pkg.ErrFmtJobDelete, id), http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}
