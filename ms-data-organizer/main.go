/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"fmt"
	"os"

	"aicsd/ms-data-organizer/clients/task_launcher"
	"aicsd/ms-data-organizer/config"
	"aicsd/ms-data-organizer/controller"
	"aicsd/pkg"
	"aicsd/pkg/auth"
	"aicsd/pkg/clients/job_handler"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/wait"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
)

func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerDataOrg))
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

	if err := service.LoadCustomConfig(&configuration.FilenameDecoder, "AttributeParser"); err != nil {
		lc.Errorf("unable to load custom writable configuration: %s", err.Error())
		os.Exit(-1)
	}
	var jwtInfo *auth.JWTInfo
	if len(configuration.PrivateKeyPath) > 0 && len(configuration.JWTAlgorithm) > 0 && len(configuration.JWTKeyPath) > 0 {
		jwtInfo, err = auth.NewToken(configuration.JWTAlgorithm, configuration.PrivateKeyPath, configuration.JWTKeyPath, configuration.JWTDuration)
		if err != nil {
			lc.Warnf("could not set jwt info: %s\nproceeding without authentication", err.Error())
		}
	}
	jobRepoClient := job_repo.NewClient(configuration.JobRepoBaseUrl, service.RequestTimeout(), jwtInfo)
	fileSenderClient := job_handler.NewClient(configuration.FileSenderBaseUrl, service.RequestTimeout(), nil)
	taskLauncherClient := task_launcher.NewClient(configuration.TaskLauncherBaseUrl, service.RequestTimeout(), jwtInfo)
	dataOrgController := controller.New(lc, jobRepoClient, fileSenderClient, taskLauncherClient, configuration.FilenameDecoder.AttributeParser, configuration.DependentServices)

	err = dataOrgController.RegisterRoutes(service)
	if err != nil {
		lc.Errorf(err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, dataOrgController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = dataOrgController.RetryOnStartup()
	if err != nil {
		lc.Error(err.Error())
	}

	err = service.Run()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	// Do any required cleanup here
	os.Exit(0)
}
