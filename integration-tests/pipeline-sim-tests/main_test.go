/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package pipeline_sim_tests

import (
	"aicsd/integration-tests/pkg"
	"aicsd/integration-tests/pkg/factory"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/google/uuid"

	"github.com/docker/docker/client"
)

var PipelineSimFactory factory.IntegrationTestFactory

func TestMain(m *testing.M) {
	// Note: Since TestMain depends on command-line flags for testing, flag.Parse() must be called explicitly.
	homeDir, _ := os.UserHomeDir()
	pkg.OemInputDir = filepath.Join(homeDir, pkg.OemInputDir)
	pkg.OemOutputDir = filepath.Join(homeDir, pkg.OemOutputDir)
	pkg.GatewayInputDir = filepath.Join(homeDir, pkg.GatewayInputDir)
	pkg.GatewayOutputDir = filepath.Join(homeDir, pkg.GatewayOutputDir)
	pkg.GatewayArchiveDir = filepath.Join(homeDir, pkg.GatewayArchiveDir)
	pkg.GatewayRejectDir = filepath.Join(homeDir, pkg.GatewayRejectDir)
	ctx, cnl := context.WithTimeout(context.Background(), 30*time.Second)
	defer cnl()
	// Create a Docker client.
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(-1)
	}
	dockerCli.NegotiateAPIVersion(ctx)

	_, err = dockerCli.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(-1)
	}
	defer cnl()

	composeFilePath := []string{"../../docker-compose-edgex.yml", "../../docker-compose-oem.yml", "../../docker-compose-gateway.yml", "../../docker-compose-sim.yml"}
	uuid := uuid.New().String()
	PipelineSimFactory = factory.NewTestFactory(composeFilePath, fmt.Sprintf("%s-%s", "integration-tests", uuid), factory.PipelineSimTestServices())
	err = PipelineSimFactory.StartAllServices()
	if err != nil {
		fmt.Print(err.Error())
	}

	returnCode := m.Run()

	err = PipelineSimFactory.Down()
	if err != nil {
		fmt.Print(err.Error())
	}
	os.Exit(returnCode)
}
