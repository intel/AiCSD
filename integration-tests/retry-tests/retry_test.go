//go:build retry_tests
// +build retry_tests

/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package retry_tests

import (
	integrationtests "aicsd/integration-tests/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/types"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileWatcherRetry is a test case that will ensure the file watcher is able to process files that have been added
// to the folder to watch while the file watcher is down
func TestFileWatcherRetry(t *testing.T) {
	fmtFileWatcherLog := "sending new file notification for file %s"
	err := PipelineSimFactory.StopServices(integrationtests.ServiceFileWatcher, integrationtests.ServiceDataOrg)
	require.NoError(t, err)
	defer PipelineSimFactory.StartAllServices()
	require.True(t, integrationtests.IsServiceDown(integrationtests.ServiceFileWatcher))
	require.True(t, integrationtests.IsServiceDown(integrationtests.ServiceDataOrg))

	sourceFile := filepath.Join(integrationtests.LocalDir, integrationtests.File1)
	destinationFile := filepath.Join(integrationtests.OemInputDir, integrationtests.File1)

	integrationtests.CopyFile(t, sourceFile, destinationFile)
	defer func() { _ = os.Remove(destinationFile) }()

	err = PipelineSimFactory.StartServices(integrationtests.ServiceFileWatcher)
	require.NoError(t, err)
	homeDir, _ := os.UserHomeDir()

	require.Eventually(t, func() bool {
		return integrationtests.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(integrationtests.ServiceFileWatcher), fmt.Sprintf("Found file %s while walking directory", strings.Replace(strings.Replace(destinationFile, homeDir+"/data/", "/tmp/", 1), "oem-", "", 1)))
	}, integrationtests.LessPauseTime, time.Second)

	require.Eventually(t, func() bool {
		return integrationtests.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(integrationtests.ServiceFileWatcher), fmt.Sprintf(fmtFileWatcherLog, strings.Replace(strings.Replace(destinationFile, homeDir+"/data/", "/tmp/", 1), "oem-", "", 1)))
	}, integrationtests.LessPauseTime, time.Second)

	sourceFile = filepath.Join(integrationtests.LocalDir, integrationtests.File2)
	destinationFile2 := filepath.Join(integrationtests.OemInputDir, integrationtests.File2)

	integrationtests.CopyFile(t, sourceFile, destinationFile2)
	defer func() { _ = os.Remove(destinationFile2) }()

	require.Eventually(t, func() bool {
		return integrationtests.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(integrationtests.ServiceFileWatcher), fmt.Sprintf("CREATE: %s", strings.Replace(strings.Replace(destinationFile2, homeDir+"/data/", "/tmp/", 1), "oem-", "", 1)))
	}, integrationtests.LessPauseTime, time.Second)

	require.Eventually(t, func() bool {
		return integrationtests.ParseServiceLog(t, PipelineSimFactory.GetServiceUrl(integrationtests.ServiceFileWatcher), fmt.Sprintf(fmtFileWatcherLog, strings.Replace(strings.Replace(destinationFile2, homeDir+"/data/", "/tmp/", 1), "oem-", "", 1)))
	}, integrationtests.LessPauseTime, time.Second)

	err = PipelineSimFactory.StopAllServices()
	require.NoError(t, err)
}

// TestResultsOutputRetry is a test case that will ensure all the services are able to execute their retry logic when
// the next service down the line is unavailable for the case where the expected output is just results
func TestResultsOutputRetry(t *testing.T) {
	// start only containers that are needed
	startNecessaryContainers(t)
	defer PipelineSimFactory.StartAllServices()

	// create the task
	taskExpect := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskId := integrationtests.POST_Task(taskExpect, &integrationtests.TaskOnlyResults, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(taskExpect, &types.Task{Id: taskId}, http.StatusOK, false)
	defer PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceTaskLauncher)

	// create the file and get the job information
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	jobId := createFileAndGetJob(t, jobRepoClient, 0, integrationtests.File1)
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})
	defer jobRepoClient.Delete(jobId)

	// continue starting and stopping services to process the job
	processInputFile(t, jobRepoClient, jobId)

	// check the job info by id - owner is none, task status may be complete, results field non empty
	assert.Eventually(t, func() bool {
		job, err := jobRepoClient.RetrieveById(jobId)
		require.NoError(t, err)
		return assert.Equal(t, pkg.OwnerNone, job.Owner) &&
			assert.Equal(t, pkg.TaskStatusComplete, job.PipelineDetails.Status) &&
			assert.Equal(t, pkg.StatusComplete, job.Status) &&
			assert.NotEmpty(t, job.PipelineDetails.Results)
	}, integrationtests.PauseTime, time.Second)
}

// TestFileOutputRetry is a test case that will ensure all the services are able to execute their retry logic when
// the next service down the line is unavailable for the case where the expected output is a file.
func TestFileOutputRetry(t *testing.T) {
	// start only containers that are needed
	startNecessaryContainers(t)
	defer PipelineSimFactory.StartAllServices()

	// create the task
	taskExpect := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskId := integrationtests.POST_Task(taskExpect, &integrationtests.TaskOnlyFile, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(taskExpect, &types.Task{Id: taskId}, http.StatusOK, false)
	defer PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceTaskLauncher)

	// create the file and get the job information
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	jobId := createFileAndGetJob(t, jobRepoClient, 0, integrationtests.File1)
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})
	defer jobRepoClient.Delete(jobId)
	defer PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceJobRepo)

	// continue starting and stopping services to process the job
	processInputFile(t, jobRepoClient, jobId)
	processOutputFile(t, jobRepoClient, jobId)

	// check the job info by id - owner is none, task status may be complete, results field non-empty
	assert.Eventually(t, func() bool {
		job, err := jobRepoClient.RetrieveById(jobId)
		require.NoError(t, err)
		return assert.Equal(t, pkg.OwnerNone, job.Owner) &&
			assert.Equal(t, pkg.TaskStatusComplete, job.PipelineDetails.Status) &&
			assert.Equal(t, pkg.StatusComplete, job.Status) &&
			assert.Equal(t, "oem", job.PipelineDetails.OutputFileHost) &&
			assert.Equal(t, pkg.FileStatusComplete, job.PipelineDetails.OutputFiles[0].Status) &&
			assert.Equal(t, pkg.CreateUserFacingError("", nil), job.PipelineDetails.OutputFiles[0].ErrorDetails) &&
			assert.Equal(t, pkg.OwnerNone, job.PipelineDetails.OutputFiles[0].Owner)
	}, integrationtests.MorePauseTime, time.Second)
}

// TestFileMultiOutputRetry is a test case that will ensure all the services are able to execute their retry logic when
// the next service down the line is unavailable for the case where the expected output is multiple files.
func TestFileMultiOutputRetry(t *testing.T) {
	// start only containers that are needed
	startNecessaryContainers(t)
	defer PipelineSimFactory.StartAllServices()

	// create task one for a multiple output file
	taskExpect := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskId := integrationtests.POST_Task(taskExpect, &integrationtests.TaskMultiFile, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(taskExpect, &types.Task{Id: taskId}, http.StatusOK, false)

	// create the file and get the job information for task one
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	jobId := createFileAndGetJob(t, jobRepoClient, 0, integrationtests.File1)
	defer jobRepoClient.Delete(jobId)
	var cleanOutputFiles []string
	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		file := fmt.Sprintf("test-image-sim%d.tiff", i)
		cleanOutputFiles = append(cleanOutputFiles, file)
	}
	defer integrationtests.CleanFiles(integrationtests.File1, cleanOutputFiles)

	time.Sleep(integrationtests.LessPauseTime)
	// continue starting and stopping services to process the job for multiple output files
	processInputFile(t, jobRepoClient, jobId)
	processOutputFile(t, jobRepoClient, jobId)

	// check the job info by id - owner is none, task status may be complete, results field nonempty
	// multiple output files
	assert.Eventually(t, func() bool {
		job, err := jobRepoClient.RetrieveById(jobId)
		require.NoError(t, err)
		return assert.Equal(t, pkg.OwnerNone, job.Owner) &&
			assert.Equal(t, pkg.TaskStatusComplete, job.PipelineDetails.Status) &&
			assert.Equal(t, pkg.StatusComplete, job.Status) &&
			assert.Equal(t, "oem", job.PipelineDetails.OutputFileHost)
	}, integrationtests.PauseTime, time.Second)

	// Note: since above check ensures the job has a complete status,
	// then the files should be in a good state below to check without using the assert.Eventually(...).
	// check output file details
	job, err := jobRepoClient.RetrieveById(jobId)
	require.NoError(t, err)
	for _, file := range job.PipelineDetails.OutputFiles {
		assert.Equal(t, pkg.FileStatusComplete, file.Status)
		assert.Equal(t, pkg.CreateUserFacingError("", nil), file.ErrorDetails)
		assert.Equal(t, pkg.OwnerNone, file.Owner)
	}

	// check that the correct number of files are written in following 3 steps:
	// step 1: check oem input file exists under /data/oem-files/input
	assert.FileExists(t, filepath.Join(integrationtests.OemInputDir, integrationtests.File1))
	// step 2: check the oem output files exist under /data/oem-files/output
	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		require.Eventually(t, func() bool {
			checkFile := filepath.Join(integrationtests.OemOutputDir, fmt.Sprintf("test-image-sim%d.tiff", i))
			return assert.FileExists(t, checkFile)
		}, integrationtests.PauseTime, time.Second)
	}
	// step 3: check the gateway archived files
	inputArchivedFilesFound, err := filepath.Glob(filepath.Join(integrationtests.GatewayArchiveDir, "*_archive_*input.tiff"))
	require.NoError(t, err)
	assert.NotNil(t, inputArchivedFilesFound)
	assert.True(t, integrationtests.Contains(inputArchivedFilesFound, filepath.Join(integrationtests.GatewayArchiveDir, "test-image_archive")))
	outputArchivedFilesFound, err := filepath.Glob(filepath.Join(integrationtests.GatewayArchiveDir, "*_archive_*output.tiff"))
	require.NoError(t, err)
	assert.NotNil(t, outputArchivedFilesFound)
	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		checkFile := filepath.Join(integrationtests.GatewayArchiveDir, fmt.Sprintf("test-image-sim%d_archive", i))
		assert.True(t, integrationtests.Contains(outputArchivedFilesFound, checkFile))
	}
}

func startNecessaryContainers(t *testing.T) {
	t.Helper()
	// change the retry timeout on the task launcher
	integrationtests.ChangeConsulKeyValue(t, integrationtests.ConsulTaskLauncherRetryUrl, "1s", true)
	// suppressing the printout with -d causes the containers to not stop :/
	err := PipelineSimFactory.StopServices(integrationtests.ServiceTaskLauncher, integrationtests.ServiceSenderOem, integrationtests.ServiceSenderGW, integrationtests.ServiceReceiverOem, integrationtests.ServiceReceiverGW, integrationtests.ServicePipelineSim)
	require.NoError(t, err)
	require.True(t, integrationtests.IsServiceDown(integrationtests.TaskLauncherUrl+integrationtests.PingEndpoint))
	require.True(t, integrationtests.IsServiceDown(integrationtests.FileSenderOEMUrl+integrationtests.PingEndpoint))
	require.True(t, integrationtests.IsServiceDown(integrationtests.FileSenderGatewayUrl+integrationtests.PingEndpoint))
	require.True(t, integrationtests.IsServiceDown(integrationtests.FileReceiverOEMUrl+integrationtests.PingEndpoint))
	require.True(t, integrationtests.IsServiceDown(integrationtests.FileReceiverGatewayUrl+integrationtests.PingEndpoint))
	require.True(t, integrationtests.IsServiceDown(integrationtests.PipelineSimUrl+integrationtests.PingEndpoint))
	// start the necessary application services
	err = PipelineSimFactory.StartServices(integrationtests.ServiceTaskLauncher)
	require.NoError(t, err)

	// check that the desired services are up and running
	deadline := time.Now().Add(integrationtests.PauseTime)
	for time.Now().Before(deadline) {
		if integrationtests.IsServiceReady(integrationtests.JobRepositoryUrl+integrationtests.PingEndpoint) && integrationtests.IsServiceReady(integrationtests.FileWatcherUrl+integrationtests.PingEndpoint) &&
			integrationtests.IsServiceReady(integrationtests.DataOraganizerUrl+integrationtests.PingEndpoint) && integrationtests.IsServiceReady(integrationtests.TaskLauncherUrl+integrationtests.PingEndpoint) &&
			integrationtests.IsServiceReady(integrationtests.ConsulUrl+integrationtests.PingEndpoint) && integrationtests.IsServiceReady(integrationtests.RedisUrl+integrationtests.PingEndpoint) {
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

}

// createFileAndGetJob is a helper function to create a file and the job created for that file. It returns the job id.
func createFileAndGetJob(t *testing.T, jobRepoClient job_repo.Client, expectedId int, file string) string {
	t.Helper()
	// set up the sample file and copy it
	sampleFile := filepath.Join(integrationtests.LocalDir, file)
	destinationFile := filepath.Join(integrationtests.OemInputDir, file)
	integrationtests.CopyFile(t, sampleFile, destinationFile)
	// get the job information
	// Wait for job to load
	require.Eventually(t, func() bool {
		jobs, err := jobRepoClient.RetrieveAll(nil)
		return len(jobs) >= 1 && err == nil
	}, integrationtests.LessPauseTime, time.Second)
	jobs, err := jobRepoClient.RetrieveAll(nil)
	require.NoError(t, err)
	// check the job information is as expected
	require.NotNil(t, jobs)
	jobId := jobs[expectedId].Id
	require.NotEmpty(t, jobId)
	require.Equal(t, pkg.OwnerDataOrg, jobs[expectedId].Owner)
	return jobId
}

// processInputFile is a helper function to stop and start services to ensure that the retry logic works across the
// services for moving an input file from oem -> gateway and producing results.
func processInputFile(t *testing.T, jobRepoClient job_repo.Client, jobId string) {
	t.Helper()
	// start the file sender oem
	err := PipelineSimFactory.StartServices(integrationtests.ServiceSenderOem)
	require.NoError(t, err)
	// check the job info by id - still owned by data-org
	integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerDataOrg)
	// call retry on data organizer
	integrationtests.CallRetry(t, integrationtests.DataOraganizerUrl)
	// check the job info by id - owner is file-sender-oem
	integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerFileSenderOem)
	err = PipelineSimFactory.StopServices(integrationtests.ServiceTaskLauncher)
	require.NoError(t, err)
	// start the file-receiver-gateway
	err = PipelineSimFactory.StartServices(integrationtests.ServiceReceiverGW)
	require.NoError(t, err)
	// This is needed due to a timing sync issue where file sender OEM calls file receiver gateway
	// before it is ready to receive requests.
	// Checking if service is down isn't enough, need to wait the full 10 seconds after stopping the service to avoid timing issues
	require.Eventually(t, func() bool {
		return integrationtests.IsServiceReady(integrationtests.FileReceiverGatewayUrl + integrationtests.PingEndpoint)
	}, integrationtests.PauseTime, time.Second)
	// call retry on the file-sender-oem
	integrationtests.CallRetry(t, integrationtests.FileSenderOEMUrl)
	// check the job info by id - owner is file-receiver-gateway, inputfile.hostname is gateway
	job := integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerFileRecvGateway)
	require.Equal(t, "gateway", job.InputFile.Hostname)
	require.Equal(t, pkg.StatusIncomplete, job.Status)
	// start the task-launcher
	err = PipelineSimFactory.StartServices(integrationtests.ServiceTaskLauncher)
	require.NoError(t, err)
	integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerFileRecvGateway)
	// call retry for file-receiver-gateway
	integrationtests.CallRetry(t, integrationtests.FileReceiverGatewayUrl)
	// check the job info by id - owner is task-launcher, status is processing, pipelineDetails.Taskid nonempty
	time.Sleep(3 * time.Second)

	job = integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerTaskLauncher)
	require.NotEmpty(t, job.PipelineDetails.TaskId)
	require.Equal(t, pkg.TaskStatusProcessing, job.PipelineDetails.Status)
	// start the pipeline simulator
	err = PipelineSimFactory.StartServices(integrationtests.ServicePipelineSim)
	require.NoError(t, err)
	// call retry endpoint on the task-launcher
	integrationtests.CallRetry(t, integrationtests.TaskLauncherUrl)
}

// processOutputFile is a helper function to stop and start services to ensure that the retry logic works across the
// services for moving an output result file from gateway -> oem.
func processOutputFile(t *testing.T, jobRepoClient job_repo.Client, jobId string) {
	t.Helper()
	// check the job id that the owner is still the task-launcher
	time.Sleep(integrationtests.LessPauseTime)
	job := integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerTaskLauncher)
	require.Equal(t, pkg.TaskStatusComplete, job.PipelineDetails.Status)
	require.Equal(t, "gateway", job.PipelineDetails.OutputFileHost)
	require.NotEmpty(t, job.PipelineDetails.OutputFiles)
	// start the file-sender-gateway
	err := PipelineSimFactory.StartServices(integrationtests.ServiceSenderGW)
	require.NoError(t, err)
	// call retry on the task-launcher
	integrationtests.CallRetry(t, integrationtests.TaskLauncherUrl)
	// check that the new owner is the file-sender-gateway
	time.Sleep(3 * time.Second)
	job = integrationtests.MatchOwner(t, jobRepoClient, jobId, pkg.OwnerFileSenderGateway)
	require.Equal(t, "gateway", job.PipelineDetails.OutputFileHost)
	require.NotEmpty(t, job.PipelineDetails.OutputFiles)
	// start the file-receiver-oem
	err = PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceReceiverOem)
	require.NoError(t, err)
	// call retry on the file-sender-gateway
	integrationtests.CallRetry(t, integrationtests.FileSenderGatewayUrl)
	time.Sleep(integrationtests.LessPauseTime)
}
