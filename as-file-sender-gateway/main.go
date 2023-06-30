/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/ms-data-organizer/clients/task_launcher"
	"aicsd/pkg/wait"
	"fmt"
	"os"

	"aicsd/as-file-sender-gateway/config"
	"aicsd/as-file-sender-gateway/controller"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
)

func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerFileSenderGateway))
	if !ok {
		os.Exit(-1)
	}

	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()
	configuration, err := config.New(service)
	if err != nil {
		lc.Errorf("failed to retrieve read app settings from configuration: %s", err.Error())
		os.Exit(-1)
	}

	jobRepoClient := job_repo.NewClient(configuration.JobRepoBaseUrl, service.RequestTimeout(), nil)

	taskRepoClient := task_launcher.NewClient(configuration.TaskLauncherBaseUrl, service.RequestTimeout(), nil)

	publisher, err := service.AddBackgroundPublisher(1)
	if err != nil {
		lc.Errorf("could not AddBackgroundPublisher: %s", err.Error())
		os.Exit(-1)
	}

	fileSenderGatewayController, err := controller.New(lc, jobRepoClient, taskRepoClient, publisher, service, configuration.FileHostname, configuration.ArchiveFolder, configuration.RejectFolder)
	if err != nil {
		lc.Errorf("failed to create controller: %s", err.Error())
		os.Exit(-1)
	}
	// Adding routes
	if err := fileSenderGatewayController.RegisterRoutes(service); err != nil {
		lc.Errorf("RegisterRoutes returned error: %s", err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, fileSenderGatewayController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	if err := fileSenderGatewayController.RetryOnStartup(); err != nil {
		lc.Errorf("failed to retry one or more jobs: %s", err.Error())
	}

	if err := service.MakeItRun(); err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	os.Exit(0)

}
