/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package main

import (
	"aicsd/ms-file-watcher/clients/data_organizer"
	"aicsd/ms-file-watcher/config"
	controller "aicsd/ms-file-watcher/controller"
	"aicsd/pkg"
	"aicsd/pkg/wait"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
)

func main() {

	service, ok := appsdk.NewAppService(fmt.Sprintf(pkg.ServiceKeyFmt, pkg.OwnerFileWatcher))
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

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	// set job to map
	dataOrgClient := data_organizer.NewClient(configuration)
	fileWatcher := controller.New(lc, dataOrgClient, configuration)

	if err := wait.ForDependencies(lc, fileWatcher.DependentServices, service.RequestTimeout()); err != nil {
		lc.Errorf("failed to wait.ForDependencies: %s", err.Error())
		os.Exit(-1)
	}

	if err := service.LoadCustomConfig(&fileWatcher.Config.App, "UpdatableSettings"); err != nil {
		lc.Errorf("unable to load custom writable configuration: %s", err.Error())
		os.Exit(-1)
	}

	// The file exclusion list is updated as a string in Consul but is stored as a slice in the Configuration
	if fileWatcher.Config.App.UpdatableSettings.FileExclusionList == "" {
		var emptySlice []string
		fileWatcher.Config.FileExclusionList = emptySlice
	} else {
		formatedExclusionList := strings.ReplaceAll(fileWatcher.Config.App.UpdatableSettings.FileExclusionList, " ", "")
		fileWatcher.Config.FileExclusionList = strings.Split(formatedExclusionList, ",")
	}

	if err := service.ListenForCustomConfigChanges(&fileWatcher.Config.App.UpdatableSettings, "UpdatableSettings", fileWatcher.ProcessConfigUpdates); err != nil {
		lc.Errorf("unable to watch custom writable configuration: %s", err.Error())
		os.Exit(-1)
	}

	wg.Add(1)
	go fileWatcher.WatchFolders(ctx, wg, fileWatcher.Config)

	err = service.MakeItRun()
	if err != nil {
		lc.Errorf("MakeItRun returned error: %s", err.Error())
		os.Exit(-1)
	}

	cancelFunc()
	wg.Wait()

	os.Exit(0)
}
