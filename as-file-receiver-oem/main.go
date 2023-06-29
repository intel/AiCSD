/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/as-file-receiver-oem/clients/file_sender"
	"aicsd/as-file-receiver-oem/config"
	"aicsd/as-file-receiver-oem/controller"
	"aicsd/as-file-receiver-oem/functions"
	"aicsd/pkg"
	"aicsd/pkg/auth"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/wait"
	"fmt"
	"os"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
)

func main() {
	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerFileRecvOem))
	if !ok {
		os.Exit(-1)
	}

	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()
	configuration, err := config.New(service)
	if err != nil {
		lc.Errorf("failed to read app settings from configuration: %s", err.Error())
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
	fileSenderClient := file_sender.NewClient(configuration.FileSenderBaseUrl, service.RequestTimeout(), jwtInfo)

	pipelineReceiver := functions.NewPipelineReceiver(jobRepoClient, fileSenderClient, configuration.FileHostname, configuration.OutputFolder)
	err = service.SetDefaultFunctionsPipeline(
		pipelineReceiver.ProcessEvent,
		pipelineReceiver.UpdateJobRepoOwner,
		pipelineReceiver.PullFile,
		pipelineReceiver.ArchiveFile,
		pipelineReceiver.UpdateJobRepoComplete,
	)
	if err != nil {
		lc.Errorf("failed to SetDefaultFunctionsPipeline: %s", err.Error())
		os.Exit(-1)
	}

	fileReceiverOEMController := controller.New(lc, jobRepoClient, fileSenderClient, configuration.FileHostname, configuration.OutputFolder, configuration.DependentServices)
	if err = wait.ForDependencies(lc, fileReceiverOEMController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	if err := fileReceiverOEMController.RetryOnStartup(); err != nil {
		lc.Errorf("failed to retry one or more jobs: %s", err.Error())
	}

	err = fileReceiverOEMController.RegisterRoutes(service)
	if err != nil {
		lc.Errorf(err.Error())
		os.Exit(-1)
	}

	if err := service.MakeItRun(); err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	os.Exit(0)
}
