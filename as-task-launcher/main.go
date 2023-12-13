/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"fmt"
	"os"

	"aicsd/pkg/clients/job_handler"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/wait"

	"aicsd/as-task-launcher/config"
	"aicsd/pkg/clients/job_repo"

	"aicsd/as-task-launcher/controller"
	"aicsd/as-task-launcher/persist"
	"aicsd/pkg"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
)

func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerTaskLauncher))
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

	secrets, err := service.SecretProvider().GetSecret(pkg.DatabasePath, "username", "password")
	if err != nil {
		lc.Errorf("failed to GetSecret for database %s: %s", pkg.DatabasePath, err.Error())
		os.Exit(-1)
	}

	redisClient := redis.NewClient(configuration.RedisHost, configuration.RedisPort, service.RequestTimeout(), secrets)
	persistence, err := persist.NewRedisDB(lc, redisClient)
	if err != nil {
		lc.Errorf("failed to connect to redis db: %s", err.Error())
		os.Exit(-1)
	}

	jobRepoClient := job_repo.NewClient(configuration.JobRepoBaseUrl, service.RequestTimeout(), nil)
	senderClient := job_handler.NewClient(configuration.FileSenderBaseUrl, service.RequestTimeout(), nil)
	publisher, err := service.AddBackgroundPublisher(1)
	if err != nil {
		lc.Errorf("failed to add background publisher: %s", err.Error())
		os.Exit(-1)
	}

	taskLauncherController := controller.New(lc, persistence, jobRepoClient, senderClient, publisher, service, configuration)
	if err = wait.ForDependencies(lc, taskLauncherController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	// Adding routes
	err = taskLauncherController.RegisterRoutes(service)
	if err != nil {
		lc.Errorf("failed to register routes: %s", err.Error())
		os.Exit(-1)
	}

	err = taskLauncherController.RetryOnStartup(configuration.RetryTimeout)
	if err != nil {
		lc.Errorf("Retry on startup failed: %s", err.Error())
	}

	if err := service.Run(); err != nil {
		lc.Errorf("Run returned error: %s", err.Error())
		os.Exit(-1)
	}

	os.Exit(0)
}
