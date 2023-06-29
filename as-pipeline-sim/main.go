/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/pkg"
	"aicsd/pkg/wait"
	"fmt"
	"os"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"

	"aicsd/as-pipeline-sim/config"
	"aicsd/as-pipeline-sim/controller"
	"aicsd/as-pipeline-sim/functions"
)

// This application service simulates an ML pipeline platform and is used by Geti and BentoML pipelines.
// There are no unit tests since it is temporary.
func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, "pipeline-sim"))
	if !ok {
		os.Exit(-1)
	}
	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()

	configuration, configErr := config.New(service)
	if configErr != nil {
		lc.Errorf("failed to retrieve read app settings from configuration: %s", configErr.Error())
		os.Exit(-1)
	}

	var err error

	justFilePipeline := functions.NewPipelineSim("", "passed", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.OnlyFilePipelineName, []string{controller.OnlyFilePipelineTopic},
		justFilePipeline.ProcessEvent,
		justFilePipeline.CreateSimulatedOutputFile,
		justFilePipeline.UpdateJobRepo,
		justFilePipeline.ReportStatus,
	)
	multiFilePipeline := functions.NewPipelineSim("", "none", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.MultiFilePipelineName, []string{controller.MultiFilePipelineTopic},
		multiFilePipeline.ProcessEvent,
		multiFilePipeline.CreateSimulatedMultiOutputFiles,
		multiFilePipeline.UpdateJobRepo,
		multiFilePipeline.ReportStatus,
	)

	justResultPipeline := functions.NewPipelineSim("CellCount, 598", "none", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.OnlyResultsPipelineName, []string{controller.OnlyResultsPipelineTopic},
		justResultPipeline.ProcessEvent,
		justResultPipeline.UpdateJobRepo,
		justResultPipeline.ReportStatus,
	)

	fileAndResultPipeline := functions.NewPipelineSim("CellCount, 101", "passed", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.FileAndResultsPipelineName, []string{controller.FileAndResultsPipelineTopic},
		fileAndResultPipeline.ProcessEvent,
		fileAndResultPipeline.CreateSimulatedOutputFile,
		fileAndResultPipeline.UpdateJobRepo,
		fileAndResultPipeline.ReportStatus,
	)

	GetiPipeline := functions.NewPipelineSim("", "passed", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.GetiPipelineName, []string{controller.GetiPipelineTopic},
		GetiPipeline.ProcessEvent,
		GetiPipeline.TriggerGetiPipeline,
		GetiPipeline.UpdateJobRepo,
		GetiPipeline.ReportStatus,
	)

	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	lc.Info("Functions Pipeline set...")

	pipelineSimController := controller.New(lc, configuration.GetiUrl)
	if err := pipelineSimController.RegisterRoutes(service); err != nil {
		lc.Errorf("RegisterRoutes returned error: %s", err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, pipelineSimController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = service.MakeItRun()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	// Do any required cleanup here
	os.Exit(0)
}
