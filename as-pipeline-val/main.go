/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"fmt"
	"os"

	"aicsd/pkg"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"

	"aicsd/as-pipeline-val/clients"
	"aicsd/as-pipeline-val/config"
	"aicsd/as-pipeline-val/controller"
	"aicsd/pkg/wait"
)

// This application service simulates an ML pipeline platform and is used by Geti and BentoML pipelines.
// There are no unit tests since it is temporary.
func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, "pipeline-val"))
	if !ok {
		os.Exit(-1)
	}

	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()
	config, err := config.New(service)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	publisher, err := service.AddBackgroundPublisher(1)
	if err != nil {
		lc.Errorf("failed to add background publisher: %s", err.Error())
		os.Exit(-1)
	}

	pipelineClient := clients.NewClient(config.PipelineUrl, service.RequestTimeout())

	kpSimController := controller.New(lc, publisher, service, config, pipelineClient)
	if err := kpSimController.RegisterRoutes(service); err != nil {
		lc.Errorf("RegisterRoutes returned error: %s", err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, kpSimController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = service.Run()
	if err != nil {
		lc.Errorf("Run returned error: %s", err.Error())
		os.Exit(-1)
	}

	// Do any required cleanup here
	os.Exit(0)
}
