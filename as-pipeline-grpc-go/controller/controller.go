/*********************************************************************
 * Copyright (c) Intel Corporation 2024
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/helpers"
	"aicsd/pkg/wait"
	"encoding/json"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/google/uuid"
	"net/http"

	"aicsd/pkg"
	"aicsd/pkg/werrors"
)

const (
	GetAvailablePipelinesRoute = "/api/v1/pipelines"
	OvmsPipelineTopic          = "ovms-grpc/yolov5"
	OvmsPipelineName           = "OVMS Pipeline"
	PipelineStatusRunning      = "Running"
)

type GrpcGoController struct {
	lc                logger.LoggingClient
	DependentServices wait.Services
	OvmsUrl           string
	GrpcClient        interface{}
}

type pipelineBody struct {
	Models []string `json:"models"`
}

func New(lc logger.LoggingClient, OvmsUrl string) *GrpcGoController {

	return &GrpcGoController{
		lc:                lc,
		DependentServices: wait.Services{wait.ServiceConsul, wait.ServiceRedis},
		OvmsUrl:           OvmsUrl,
	}
}

func (p *GrpcGoController) RegisterRoutes(service interfaces.ApplicationService) error {
	if err := service.AddRoute(GetAvailablePipelinesRoute, p.getPipelinesHandler, http.MethodGet); err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, GetAvailablePipelinesRoute)
	}

	if err := service.AddRoute(GetAvailablePipelinesRoute, p.getPipelinesHandler, http.MethodGet); err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, GetAvailablePipelinesRoute)
	}

	p.lc.Info("Routes added...")
	return nil
}

func (p *GrpcGoController) getPipelinesHandler(writer http.ResponseWriter, _ *http.Request) {
	pipelines := []struct {
		Id                string `json:"id"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		SubscriptionTopic string `json:"subscriptionTopic"`
		Status            string `json:"status"`
	}{
		{
			Id:                uuid.NewString(),
			Name:              OvmsPipelineName,
			Description:       "Pipeline that calls OVMS yolov5",
			SubscriptionTopic: OvmsPipelineTopic,
			Status:            PipelineStatusRunning,
		},
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
