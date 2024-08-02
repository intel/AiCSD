/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/helpers"
	"aicsd/pkg/wait"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/google/uuid"

	"aicsd/pkg"
	"aicsd/pkg/werrors"
)

const (
	GetAvailablePipelinesRoute  = "/api/v1/pipelines"
	OnlyFilePipelineName        = "OnlyFile"
	OnlyFilePipelineTopic       = "only-file"
	MultiFilePipelineName       = "MultiFile"
	MultiFilePipelineTopic      = "multi-file"
	OnlyResultsPipelineName     = "OnlyResults"
	OnlyResultsPipelineTopic    = "only-results"
	FileAndResultsPipelineName  = "FileAndResults"
	FileAndResultsPipelineTopic = "file-and-results"
	GetiPipelineName            = "EvamPipeline"
	GetiPipelineTopic           = "geti/#"
	OvmsPipelineName            = "OVMS Pipeline"
	PipelineStatusRunning       = "Running"

	EVAMPipelines = "/pipelines"
)

type PipelineSimController struct {
	lc                logger.LoggingClient
	DependentServices wait.Services
	GetiUrl           string
}

type EvamBodyPipelines struct {
	Version string `json:"version"`
}

func New(lc logger.LoggingClient, GetiUrl string) *PipelineSimController {
	return &PipelineSimController{
		lc: lc,
		DependentServices: wait.Services{wait.ServiceConsul, wait.ServiceRedis, wait.ServiceJobRepo,
			wait.ServiceTaskLauncher},
		GetiUrl: GetiUrl,
	}
}

func (p *PipelineSimController) RegisterRoutes(service interfaces.ApplicationService) error {
	if err := service.AddRoute(GetAvailablePipelinesRoute, p.getPipelinesHandler, http.MethodGet); err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, GetAvailablePipelinesRoute)
	}

	p.lc.Info("Routes added...")
	return nil
}

func (p *PipelineSimController) getPipelinesHandler(writer http.ResponseWriter, _ *http.Request) {
	pipelines := []struct {
		Id                string `json:"id"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		SubscriptionTopic string `json:"subscriptionTopic"`
		Status            string `json:"status"`
	}{
		{
			Id:                uuid.NewString(),
			Name:              OnlyFilePipelineName,
			Description:       "Pipeline that generates only an output file",
			SubscriptionTopic: OnlyFilePipelineTopic,
			Status:            PipelineStatusRunning,
		},
		{
			Id:                uuid.NewString(),
			Name:              MultiFilePipelineName,
			Description:       "Pipeline that generates multiple output files",
			SubscriptionTopic: MultiFilePipelineTopic,
			Status:            PipelineStatusRunning,
		},
		{
			Id:                uuid.NewString(),
			Name:              OnlyResultsPipelineName,
			Description:       "Pipeline that generates only results",
			SubscriptionTopic: OnlyResultsPipelineTopic,
			Status:            PipelineStatusRunning,
		},
		{
			Id:                uuid.NewString(),
			Name:              FileAndResultsPipelineName,
			Description:       "Pipeline that generates output file and results",
			SubscriptionTopic: FileAndResultsPipelineTopic,
			Status:            PipelineStatusRunning,
		},
	}

	// Get available models from EVAM
	getiURL := fmt.Sprintf("%s%s", p.GetiUrl, EVAMPipelines)

	resp, err := http.Get(getiURL)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			err = werrors.WrapMsgf(err, "unable to read data from EVAM")
			helpers.HandleErrorMessage(p.lc, writer, err, http.StatusInternalServerError)
		}

		var EvamPipelines []EvamBodyPipelines
		err = json.Unmarshal(body, &EvamPipelines)
		if err != nil {
			err = werrors.WrapMsgf(err, "unable to unmarshal data from EVAM")
			helpers.HandleErrorMessage(p.lc, writer, err, http.StatusInternalServerError)
		}

		for _, value := range EvamPipelines {

			//process & add EVAM pipelines
			pipeline := struct {
				Id                string `json:"id"`
				Name              string `json:"name"`
				Description       string `json:"description"`
				SubscriptionTopic string `json:"subscriptionTopic"`
				Status            string `json:"status"`
			}{
				Id:                uuid.NewString(),
				Name:              value.Version + " " + GetiPipelineName,
				Description:       "Pipeline that calls EVAM for " + value.Version,
				SubscriptionTopic: "geti/" + value.Version,
				Status:            PipelineStatusRunning,
			}
			pipelines = append(pipelines, pipeline)
		}

		defer resp.Body.Close()
	}

	data, err := json.Marshal(pipelines)
	if err != nil {
		err = werrors.WrapMsgf(err, "unable to marshal available pipelines")
		helpers.HandleErrorMessage(p.lc, writer, err, http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(data)
	if err != nil {
		err = werrors.WrapMsgf(err, "unable to write available pipelines response")
		helpers.HandleErrorMessage(p.lc, writer, err, http.StatusInternalServerError)
		return
	}
}
