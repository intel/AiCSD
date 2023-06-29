/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/pkg/auth"
	"aicsd/pkg/wait"
	"fmt"
	"os"

	"aicsd/ms-file-sender-oem/clients/file_receiver"
	"aicsd/ms-file-sender-oem/config"
	"aicsd/ms-file-sender-oem/controller"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
)

func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerFileSenderOem))
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

	var jwtInfo *auth.JWTInfo
	if len(configuration.PrivateKeyPath) > 0 && len(configuration.JWTAlgorithm) > 0 && len(configuration.JWTKeyPath) > 0 {
		jwtInfo, err = auth.NewToken(configuration.JWTAlgorithm, configuration.PrivateKeyPath, configuration.JWTKeyPath, configuration.JWTDuration)
		if err != nil {
			lc.Warnf("could not set jwt info: %s\nproceeding without authentication", err.Error())
		}
	}

	dataRepoClient := job_repo.NewClient(configuration.JobRepoBaseUrl, service.RequestTimeout(), jwtInfo)
	fileReceiverClient := file_receiver.NewClient(configuration.FileReceiverBaseUrl, service.RequestTimeout(), configuration.RetryAttempts, configuration.RetryWaitTime, jwtInfo)
	fileSender := controller.New(lc, dataRepoClient, fileReceiverClient, configuration.FileHostname, configuration.DependentServices)
	err = fileSender.RegisterRoutes(service)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, fileSender.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = fileSender.RetryOnStartup()
	if err != nil {
		lc.Errorf("Retry on startup failed: %s", err.Error())
	}

	err = service.MakeItRun()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	os.Exit(0)
}
