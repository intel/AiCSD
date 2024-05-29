/*********************************************************************
 * Copyright (c) Intel Corporation 2024
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/pkg"
	"aicsd/pkg/wait"
	"fmt"
	"net/http"
	"os"
	"strings"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"

	"aicsd/as-pipeline-grpc-go/config"
	"aicsd/as-pipeline-grpc-go/controller"
	"aicsd/as-pipeline-grpc-go/functions"
)

// This application service calls an OVMS ML pipeline using a grpc call
func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, "pipeline-grpc-go"))
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

	lc.Debugf("main: OvmsGrpcUrl set to: %s", configuration.OvmsUrl)

	var err error

	grpcGoPipeline := functions.NewPipelineGrpcGo("", "passed", configuration)
	err = service.AddFunctionsPipelineForTopics(controller.OvmsPipelineName, []string{controller.OvmsPipelineTopic},
		grpcGoPipeline.ProcessEvent,
		grpcGoPipeline.SetupStreamConfig,
		grpcGoPipeline.RunOvmsModel,
		grpcGoPipeline.UpdateJobRepo,
		grpcGoPipeline.ReportStatus,
	)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	lc.Info("Functions Pipeline set...")

	grpcGoController := controller.New(lc, configuration.OvmsUrl)
	if err := grpcGoController.RegisterRoutes(service); err != nil {
		lc.Errorf("RegisterRoutes returned error: %s", err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, grpcGoController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	// set up the stream to output data - this will only work for one pipeline at a time
	http.Handle("/", grpcGoPipeline.CvParams.Stream)
	httpErr := http.ListenAndServe(configuration.OutputStreamHost, nil)
	if strings.Contains(httpErr.Error(), "address already in use") {
		fmt.Println("ERROR(5)- http server binding issue: ", httpErr.Error())
		os.Exit(5)
	}

	err = service.MakeItRun()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	// Do any required cleanup here
	grpcGoPipeline.CloseGrpcConnection()

	os.Exit(0)
}
