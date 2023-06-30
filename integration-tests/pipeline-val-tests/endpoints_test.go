/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package pipeline_val_tests

import (
	simTypes "aicsd/as-pipeline-val/types"
	integrationtests "aicsd/integration-tests/pkg"
	"aicsd/pkg"
	"net/http"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobQuery tests the Jobs endpoint on the pipeline validator service
func TestJobQuery(t *testing.T) {
	e := httpexpect.Default(t, integrationtests.PipelineValUrl)

	req := e.GET(pkg.EndpointJob).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors)

	resp := req.Expect()
	resp.Status(http.StatusOK)
	resp.Body().Equal("[]")
}

// TestGetPipelines tests the GetPipelines endpoint on the pipeline validator service
func TestGetPipelines(t *testing.T) {
	e := httpexpect.Default(t, integrationtests.PipelineValUrl)

	req := e.GET(pkg.EndpointGetPipelines).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors)

	resp := req.Expect()
	resp.Status(http.StatusOK)
	resp.Body().NotEmpty()
}

// TestLaunchPipeline ensures that an event may be created and processed using the pipeline validator and pipeline simulator
func TestLaunchPipeline(t *testing.T) {
	// copy and clean up files
	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.GatewayInputDir, integrationtests.File1))
	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.GatewayInputDir, integrationtests.File1))
	}, integrationtests.PauseTime, time.Second)
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})
	// call LaunchPipeline
	inputInfo := simTypes.LaunchInfo{
		InputFileLocation: path.Join(filepath.Join("/tmp", "files", "input"), integrationtests.File1),
		PipelineTopic:     "only-file",
		OutputFileFolder:  filepath.Join("/tmp", "files", "output"),
		ModelParams:       make(map[string]string),
	}
	e := httpexpect.Default(t, integrationtests.PipelineValUrl)
	e.POST(pkg.EndpointLaunchPipeline).
		WithJSON(inputInfo).
		Expect().
		Status(http.StatusCreated)
	// verify output
	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.GatewayOutputDir, integrationtests.File1out))
	}, integrationtests.PauseTime, time.Second)

	e.GET(pkg.EndpointJob).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusOK).
		Body().
		Contains(pkg.TaskStatusComplete)
}
