/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/pkg/translation"
	"aicsd/pkg/wait"
	"fmt"
	"os"

	"aicsd/ms-job-repository/config"
	"aicsd/ms-job-repository/controller"
	"aicsd/ms-job-repository/persist"
	"aicsd/pkg"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/werrors"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
)

func main() {
	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.JobRepository))
	if !ok {
		os.Exit(-1)
	}
	// Leverage the built-in logging service in EdgeX
	lc := service.LoggingClient()
	configuration, err := config.New(service)
	if err != nil {
		lc.Errorf(werrors.WrapErr(err, pkg.ErrLoadingConfig).Error())
		os.Exit(-1)
	}
	bundle, err := translation.NewBundler(configuration.LocalizationFiles)
	if err != nil {
		lc.Errorf("failed to create NewBundler for localization: %s", err.Error())
		os.Exit(-1)
	}

	secrets, err := service.GetSecret(pkg.DatabasePath, "username", "password")
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

	dataRepoController := controller.New(lc, persistence, bundle)

	err = dataRepoController.RegisterRoutes(service)
	if err != nil {
		lc.Error(err.Error())
		os.Exit(-1)
	}

	if err = wait.ForDependencies(lc, dataRepoController.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	err = service.MakeItRun()
	if err != nil {
		lc.Errorf(werrors.WrapErr(err, pkg.ErrRunningService).Error())
		os.Exit(-1)
	}

	// Do any required cleanup here
	err = persistence.Disconnect()
	if err != nil {
		lc.Error(werrors.WrapMsg(err, "could not disconnect from persistence").Error())
	}
	os.Exit(0)
}
