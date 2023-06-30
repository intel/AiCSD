/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pipeline_sim_tests

import (
	"aicsd/integration-tests/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExclusionPass tests a file-watcher that has an empty file-exclusion list,
// which means that every new input file will get processed in testing.
func TestExclusionPass(t *testing.T) {
	jobRepoClient := job_repo.NewClient(pkg.JobRepositoryUrl, pkg.HttpTimeout, nil)
	defer pkg.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(pkg.File1, "only-file")
	e := httpexpect.Default(t, pkg.TaskLauncherUrl)
	taskObj.Id = pkg.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer pkg.DELETE_Task(e, &taskObj, http.StatusOK, false)

	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File1), path.Join(pkg.OemInputDir, pkg.File1))
	defer pkg.CleanFiles(pkg.File1, []string{pkg.File1out})

	assert.FileExists(t, path.Join(pkg.OemInputDir, pkg.File1))

	destinationFile := filepath.Join(pkg.OemInputDir, pkg.File1)
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return pkg.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(pkg.ServiceFileWatcher), fmt.Sprintf("CREATE: %s", strings.Replace(strings.Replace(destinationFile, homeDir+"/data/", "/tmp/", 1), "oem-", "", 1)))
	}, pkg.LessPauseTime, time.Second)

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(pkg.OemOutputDir, pkg.File1out))
	}, pkg.PauseTime, time.Second)
}

// TestExclusionCatch tests a file-watcher that has a string in the file-exclusion list,
// which means that the input files will get excluded if their filenames contain strings
// found in the file-exclusion list and therefore will not be processed.
func TestExclusionCatch(t *testing.T) {
	jobRepoClient := job_repo.NewClient(pkg.JobRepositoryUrl, pkg.HttpTimeout, nil)
	defer pkg.CleanJobs(t, jobRepoClient)
	value := "-2"
	pkg.ChangeConsulKeyValue(t, fmt.Sprintf(pkg.ConsulChangeFWConfigVarUrl, pkg.FileExclusionList), value, true)
	defer pkg.ChangeConsulKeyValue(t, fmt.Sprintf(pkg.ConsulChangeFWConfigVarUrl, pkg.FileExclusionList), "", true)

	taskObjPass := helpers.CreateTestTask(pkg.File1, "only-file")
	taskObjBlock := helpers.CreateTestTask(pkg.File2, "only-file")
	e := httpexpect.Default(t, pkg.TaskLauncherUrl)
	taskObjPass.Id = pkg.POST_Task(e, &taskObjPass, http.StatusCreated, false)
	taskObjBlock.Id = pkg.POST_Task(e, &taskObjBlock, http.StatusCreated, false)
	defer pkg.DELETE_Task(e, &taskObjPass, http.StatusOK, false)
	defer pkg.DELETE_Task(e, &taskObjBlock, http.StatusOK, false)

	require.Eventually(t, func() bool {
		return pkg.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(pkg.ServiceFileWatcher), fmt.Sprintf("File Exclusion List set to: %s", value))
	}, pkg.LessPauseTime, time.Second)

	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File1), path.Join(pkg.OemInputDir, pkg.File1))
	defer pkg.CleanFiles(pkg.File1, []string{pkg.File1out})
	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File2), path.Join(pkg.OemInputDir, pkg.File2))
	defer pkg.CleanFiles(pkg.File2, []string{pkg.File2out})

	require.FileExists(t, path.Join(pkg.OemInputDir, pkg.File1))
	require.FileExists(t, path.Join(pkg.OemInputDir, pkg.File2))

	require.Eventually(t, func() bool {
		return assert.NoFileExists(t, path.Join(pkg.OemOutputDir, pkg.File2out)) && assert.FileExists(t, path.Join(pkg.OemOutputDir, pkg.File1out))
	}, pkg.PauseTime, time.Second)
}

// TestFlatFileStructure tests the file-watcher, sets the configuration to use flat
// file-structure and drops in a nested file.
// The expected behavior is that the nested file is not processed.
func TestFlatFileStructure(t *testing.T) {
	jobRepoClient := job_repo.NewClient(pkg.JobRepositoryUrl, pkg.HttpTimeout, nil)
	defer pkg.CleanJobs(t, jobRepoClient)

	pkg.ChangeConsulKeyValue(t, fmt.Sprintf(pkg.ConsulChangeFWConfigVarUrl, pkg.WatchSubfolders), "false", true)
	defer pkg.ChangeConsulKeyValue(t, fmt.Sprintf(pkg.ConsulChangeFWConfigVarUrl, pkg.WatchSubfolders), "true", true)

	taskObj1 := helpers.CreateTestTask(pkg.File1, "only-file")
	taskObj2 := helpers.CreateTestTask(pkg.File2, "only-file")
	e := httpexpect.Default(t, pkg.TaskLauncherUrl)
	taskObj1.Id = pkg.POST_Task(e, &taskObj1, http.StatusCreated, false)
	taskObj2.Id = pkg.POST_Task(e, &taskObj2, http.StatusCreated, false)
	defer pkg.DELETE_Task(e, &taskObj1, http.StatusOK, false)
	defer pkg.DELETE_Task(e, &taskObj2, http.StatusOK, false)

	fileInputDir := filepath.Join(pkg.OemInputDir, "t0")
	err := os.Mkdir(fileInputDir, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(fileInputDir)

	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File1), path.Join(pkg.OemInputDir, pkg.File1))
	defer pkg.CleanFiles(pkg.File1, []string{pkg.File1out})
	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File2), path.Join(fileInputDir, pkg.File2))
	defer pkg.CleanFiles(pkg.File2, []string{pkg.File2out})

	assert.FileExists(t, path.Join(pkg.OemInputDir, pkg.File1))
	assert.FileExists(t, path.Join(fileInputDir, pkg.File2))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(pkg.OemOutputDir, pkg.File1out)) && assert.NoFileExists(t, path.Join(pkg.OemOutputDir, pkg.File2out))
	}, pkg.PauseTime, time.Second)
}

// TestSubfolderStructure tests the file-watcher, using the default configuration of a nested
// file-structure, and drops in a nested file.
// The expected behavior is that the nested file is processed.
func TestSubfolderStructure(t *testing.T) {
	jobRepoClient := job_repo.NewClient(pkg.JobRepositoryUrl, pkg.HttpTimeout, nil)
	defer pkg.CleanJobs(t, jobRepoClient)

	// TODO: Figure out why this is necessary (since its not waiting on any specific action taken)
	// TODO: change to track something rather than wait an arbitrary amount of time
	// Note this fixes a timing issue pending system
	time.Sleep(time.Second * 3)

	taskObj := helpers.CreateTestTask(pkg.File3, "only-file")
	e := httpexpect.Default(t, pkg.TaskLauncherUrl)
	taskObj.Id = pkg.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer pkg.DELETE_Task(e, &taskObj, http.StatusOK, false)

	fileInputDir := filepath.Join(pkg.OemInputDir, "t0")
	err := os.Mkdir(fileInputDir, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Join(pkg.GatewayInputDir, "t0"))
	defer os.RemoveAll(fileInputDir)

	pkg.CopyFile(t, path.Join(pkg.LocalDir, pkg.File3), path.Join(fileInputDir, pkg.File3))
	defer pkg.CleanFiles(pkg.File3, []string{pkg.File3out})

	assert.FileExists(t, path.Join(fileInputDir, pkg.File3))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(pkg.OemOutputDir, pkg.File3out))
	}, pkg.PauseTime, time.Second)
}
