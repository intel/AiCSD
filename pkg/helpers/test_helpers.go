/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

// This file is designed to add test helper functions.
// While not ideal, it is acknowledged that they do get added to the resulting go binaries.

package helpers

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aicsd/pkg"
	"aicsd/pkg/types"
)

func createTestFileInfo(fileHostName string) types.FileInfo {
	return types.FileInfo{
		Hostname:  fileHostName,
		DirName:   filepath.Join(".", "test"),
		Name:      "test-image.tiff",
		Extension: ".tiff",
		Attributes: map[string]string{
			"LabName":      "ScienceLab",
			"LabEquipment": "Microscope",
			"Operator":     "Scientist 1",
		},
	}
}

// CreateTestFile creates an OutputFile to use for test files
func CreateTestFile(name, status, owner string, errDetails *pkg.UserFacingError) types.OutputFile {
	var outFile types.OutputFile
	outFile.DirName = filepath.Join("test")
	outFile.Name = name
	outFile.Extension = ".tiff"
	outFile.ArchiveName = mock.Anything
	outFile.Status = status
	if errDetails != nil && errDetails.Owner != "" {
		outFile.ErrorDetails = errDetails
	} else {
		outFile.ErrorDetails = pkg.CreateUserFacingError("", nil)
	}
	outFile.Owner = owner
	return outFile
}

// CreateTestJob creates a job to use for testing purposes
func CreateTestJob(owner string, hostname string) types.Job {
	return types.Job{
		Id:        "1",
		Owner:     owner,
		InputFile: createTestFileInfo(hostname),
		PipelineDetails: types.PipelineInfo{
			TaskId:         "1",
			Status:         pkg.TaskStatusComplete,
			OutputFileHost: hostname,
			OutputFiles:    []types.OutputFile{CreateTestFile("outfile1.tiff", pkg.FileStatusIncomplete, owner, pkg.CreateUserFacingError("", nil))},
			Results:        "count,3",
		},
		LastUpdated:  time.Now().UTC().UnixNano(),
		ErrorDetails: pkg.CreateUserFacingError("", nil),
	}
}

func CreateTestTask(file string, pipelineId string) types.Task {
	return types.Task{
		Description:      "Generate Result",
		JobSelector:      "{ \"==\" : [ { \"var\" : \"InputFile.Name\" }, \"" + file + "\" ] }",
		PipelineId:       pipelineId,
		ResultFileFolder: "/tmp/files/output",
		ModelParameters: map[string]string{
			"Brightness": "0",
		},
		LastUpdated: 0,
	}
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// SetupTestFiles is a helper to create any test directories/files needed.
// It returns a clean-up function that must be called in a subsequent defer for cleaning up test resources.
func SetupTestFiles(t *testing.T) func(t *testing.T) {
	t.Helper()
	// make test directory specifically for writing output files
	const (
		testDir    = "test"
		archiveDir = "archive"
		file0      = "test-image.tiff"
		file1      = "outfile1.tiff"
		file2      = "outfile2.tiff"
	)
	require.NoError(t, os.Mkdir(filepath.Join(".", testDir), pkg.FolderPermissions))
	require.NoError(t, os.Mkdir(filepath.Join(".", testDir, archiveDir), pkg.FolderPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(".", testDir, file0), []byte{}, pkg.FilePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(".", testDir, file1), []byte{}, pkg.FilePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(".", testDir, file2), []byte{}, pkg.FilePermissions))

	return func(t *testing.T) {
		t.Logf("tearing down test files for test case named: %s", t.Name())
		// test file clean up
		require.NoError(t, os.RemoveAll(testDir))
		require.NoError(t, os.RemoveAll(archiveDir))
	}
}
