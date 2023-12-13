/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"fmt"
	"os"

	"aicsd/pkg/wait"

	"aicsd/ms-file-receiver-gateway/config"
	"aicsd/ms-file-receiver-gateway/controller"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_handler"
	"aicsd/pkg/clients/job_repo"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
)

func main() {
	serviceKey := fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerFileRecvGateway)

	service, ok := appsdk.NewAppService(serviceKey)
	if !ok {
		os.Exit(-1)
	}

	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()

	configuration, err := config.New(service)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	jobRepoClient := job_repo.NewClient(configuration.JobRepoBaseUrl, service.RequestTimeout(), nil)
	taskLauncherClient := job_handler.NewClient(configuration.TaskLauncherBaseUrl, service.RequestTimeout(), nil)

	// set job to map
	fileReceiver := controller.New(lc, jobRepoClient, taskLauncherClient, configuration)
	err = fileReceiver.RegisterRoutes(service)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, fileReceiver.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = fileReceiver.RetryOnStartup()
	if err != nil {
		lc.Error(err.Error())
	}

	err = service.Run()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	os.Exit(0)
}
