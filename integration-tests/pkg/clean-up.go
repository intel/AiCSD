/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import (
	"aicsd/pkg/clients/job_repo"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// CleanJobs is a helper function to clean up all of the jobs from the job repo
func CleanJobs(t *testing.T, jobRepoClient job_repo.Client) {
	t.Helper()

	jobs, err := jobRepoClient.RetrieveAll(nil)
	require.NoError(t, err)
	require.NotNil(t, jobs)

	for _, job := range jobs {
		jobRepoClient.Delete(job.Id)
	}
}

// CleanFiles is a helper function to clean up all the possible test files
// Use []string{...} for singular and multiple files
func CleanFiles(inputFile string, outputFiles []string) {
	os.Remove(path.Join(OemInputDir, inputFile))
	for _, file := range outputFiles {
		os.Remove(path.Join(OemOutputDir, file))
		os.Remove(path.Join(GatewayOutputDir, file))
	}
	os.Remove(path.Join(GatewayInputDir, inputFile))
	inputFilename := strings.Split(inputFile, ".")[0]
	files, err := os.ReadDir(GatewayArchiveDir)
	if err != nil {
		return
	}
	for _, file := range files {
		if !file.IsDir() {
			// remove input file
			if strings.Contains(file.Name(), inputFilename) {
				os.Remove(path.Join(GatewayArchiveDir, file.Name()))
			}
		}
	}
}
