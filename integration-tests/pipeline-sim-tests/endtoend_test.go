/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pipeline_sim_tests

import (
	"aicsd/as-file-receiver-oem/clients/file_sender"
	integrationtests "aicsd/integration-tests/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"bytes"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileOutputPositive performs end to end integration testing where the expected output from the system is an output file.
// This also performs checks with the Accept-Language header set to support Chinese,
// and ensures that job/file related fields are translated accordingly.
func TestFileOutputPositive(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File1, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.OemInputDir, integrationtests.File1))
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File1))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File1out))
	}, integrationtests.MorePauseTime, time.Second)

	// verify job and file output results are as expected in English
	jobs, err := jobRepoClient.RetrieveAll(nil)
	require.NoError(t, err)
	assert.NotNil(t, jobs)
	assert.NotEmpty(t, jobs[0].InputFile.ArchiveName)
	assert.Equal(t, pkg.StatusComplete, jobs[0].Status)
	assert.Equal(t, pkg.FileStatusComplete, jobs[0].PipelineDetails.OutputFiles[0].Status)
	assert.NotEmpty(t, jobs[0].PipelineDetails.OutputFiles[0].ArchiveName)
	assert.Equal(t, pkg.CreateUserFacingError("", nil), jobs[0].PipelineDetails.OutputFiles[0].ErrorDetails)
	assert.Equal(t, pkg.OwnerNone, jobs[0].PipelineDetails.OutputFiles[0].Owner)

	// verify job and file output results are as expected translated to Chinese
	integrationtests.AssertInternationalization(t, jobRepoClient, pkg.StatusComplete, pkg.FileStatusComplete, pkg.OwnerNone, pkg.OwnerNone,
		pkg.CreateUserFacingError("", nil), pkg.CreateUserFacingError("", nil))
}

// TestFileOutputNegative performs end to end integration testing where the expected output from the system is an output file.
// This function checks a job and file error case, along with the corresponding field internationalization.
func TestFileOutputNegative(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File1, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	err := PipelineSimFactory.StopServices(integrationtests.ServiceReceiverOem)
	require.NoError(t, err)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.OemInputDir, integrationtests.File1))
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File1))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.GatewayOutputDir, integrationtests.File1out))
	}, integrationtests.MorePauseTime, time.Second)

	assert.NoError(t, os.Remove(path.Join(integrationtests.GatewayOutputDir, integrationtests.File1out)))

	fileSenderClient := file_sender.NewClient(integrationtests.FileSenderGatewayUrl, integrationtests.HttpTimeout, nil)

	err = PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceReceiverOem)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return integrationtests.IsServiceReady(integrationtests.FileReceiverOEMUrl + integrationtests.PingEndpoint)
	}, integrationtests.PauseTime, time.Second)
	err = fileSenderClient.Retry()
	require.NoError(t, err)

	// verify job repo has output file present to check values of
	require.Eventually(t, func() bool {
		jobs, err := jobRepoClient.RetrieveAll(nil)
		require.NoError(t, err)
		assert.NotNil(t, jobs)
		if len(jobs[0].PipelineDetails.OutputFiles) > 0 {
			return true
		}
		return false
	}, integrationtests.MorePauseTime, time.Second)

	require.Eventually(t, func() bool {
		jobs, err := jobRepoClient.RetrieveAll(nil)
		require.NoError(t, err)
		success := assert.NotNil(t, jobs) &&
			assert.Equal(t, pkg.StatusFileError, jobs[0].Status) &&
			assert.Equal(t, pkg.FileStatusInvalid, jobs[0].PipelineDetails.OutputFiles[0].Status) &&
			assert.Equal(t, pkg.OwnerNone, jobs[0].Owner) &&
			assert.NotEmpty(t, jobs[0].InputFile.ArchiveName) &&
			assert.Equal(t, pkg.OwnerFileSenderGateway, jobs[0].PipelineDetails.OutputFiles[0].Owner) &&
			assert.Equal(t, pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting), jobs[0].ErrorDetails) &&
			assert.Equal(t, pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileInvalid), jobs[0].PipelineDetails.OutputFiles[0].ErrorDetails)
		if !success {
			err := PipelineSimFactory.StopServices(integrationtests.ServiceReceiverOem)
			require.NoError(t, err)
			err = PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceReceiverOem)
			require.NoError(t, err)
			err = fileSenderClient.Retry()
			require.NoError(t, err)
		}
		return success
	}, integrationtests.PauseTime, time.Second)

	// verify job and file output results are as expected translated to Chinese
	integrationtests.AssertInternationalization(t, jobRepoClient, pkg.StatusFileError, pkg.FileStatusInvalid, pkg.OwnerNone, pkg.OwnerFileSenderGateway,
		pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting), pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileInvalid))
}

// TestResultsOutput performs end to end integration testing where the expected output from the system is a filled in results field
func TestResultsOutput(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File4, "only-results")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File4), path.Join(integrationtests.OemInputDir, integrationtests.File4))
	defer integrationtests.CleanFiles(integrationtests.File4, []string{integrationtests.File4out})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File4))

	require.Eventually(t, func() bool {
		return assert.NoFileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File4out))
	}, integrationtests.PauseTime, time.Second)
}

// TestNoMatchingTask ensures that the processing of the file stops if there are no matching tasks
func TestNoMatchingTask(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask("fake-image.tiff", "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File5), path.Join(integrationtests.OemInputDir, integrationtests.File5))
	defer integrationtests.CleanFiles(integrationtests.File5, []string{integrationtests.File5out})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File5))

	require.Eventually(t, func() bool {
		return assert.NoFileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File5out))
	}, integrationtests.PauseTime, time.Second)
}

// TestMultipleInputFiles ensures that the system can handle the processing of simultaneously dropped files.
func TestMultipleInputFiles(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File1, "only-file")
	taskObj2 := helpers.CreateTestTask(integrationtests.File2, "only-file")
	taskObj3 := helpers.CreateTestTask(integrationtests.File3, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)
	taskObj2.Id = integrationtests.POST_Task(e, &taskObj2, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj2, http.StatusOK, false)
	taskObj3.Id = integrationtests.POST_Task(e, &taskObj3, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj3, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File2), path.Join(integrationtests.OemInputDir, integrationtests.File2))
	defer integrationtests.CleanFiles(integrationtests.File2, []string{integrationtests.File2out})
	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File3), path.Join(integrationtests.OemInputDir, integrationtests.File3))
	defer integrationtests.CleanFiles(integrationtests.File3, []string{integrationtests.File3out})
	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.OemInputDir, integrationtests.File1))
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File1))
	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File2))
	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.File3))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File1out))
	}, integrationtests.PauseTime, time.Second)
	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File2out))
	}, integrationtests.PauseTime, time.Second)
	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File3out))
	}, integrationtests.PauseTime, time.Second)
}

// TestFileSizes ensures that the system can handle files of varying sizes up to 2.4MB.
func TestFileSizes(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.FileHiDef, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.FileHiDef), path.Join(integrationtests.OemInputDir, integrationtests.FileHiDef))
	defer integrationtests.CleanFiles(integrationtests.FileHiDef, []string{integrationtests.FileHiDefout})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.FileHiDef))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.FileHiDefout))
	}, integrationtests.PauseTime, time.Second)
}

// TestArchival ensures that job(s) input/output files get archived on the GW after the files get copied to the OEM machine.
func TestArchival(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File1, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)

	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.OemInputDir, integrationtests.File1))
	defer integrationtests.CleanFiles(integrationtests.File1, []string{integrationtests.File1out})

	fileMoved, err := helpers.Exists(path.Join(integrationtests.OemInputDir, integrationtests.File1))
	require.NoError(t, err)
	require.True(t, fileMoved)

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.File1out))
	}, integrationtests.PauseTime, time.Second)

	// verify there are archive files suffixed with input for file1
	inputArchivedFilesFound, err := filepath.Glob(filepath.Join(integrationtests.GatewayArchiveDir, "*_archive_*input.tiff"))
	require.NoError(t, err)
	assert.NotNil(t, inputArchivedFilesFound)
	assert.True(t, integrationtests.Contains(inputArchivedFilesFound, filepath.Join(integrationtests.GatewayArchiveDir, integrationtests.File1archive)))

	// Allow time in case of multiple output files for them to be archived properly and
	// verify there are archive files suffixed with output for file1
	outputArchivedFilesFound, err := filepath.Glob(path.Join(integrationtests.GatewayArchiveDir, "*_archive_*output.tiff"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return assert.NotNil(t, outputArchivedFilesFound) &&
			assert.True(t, integrationtests.Contains(outputArchivedFilesFound, filepath.Join(integrationtests.GatewayArchiveDir, integrationtests.File1outArchive)))
	}, integrationtests.PauseTime, time.Second)
}

// TestMultipleOutputFiles ensures that the system can handle the processing of simultaneously dropped files,
// and their archival process.
func TestMultipleOutputFiles(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.File1, "multi-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	var cleanOutputFiles []string
	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		file := fmt.Sprintf("test-image-sim%d.tiff", i)
		cleanOutputFiles = append(cleanOutputFiles, file)
	}

	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), filepath.Join(integrationtests.OemInputDir, integrationtests.File1))
	defer integrationtests.CleanFiles(integrationtests.File1, cleanOutputFiles)
	assert.FileExists(t, filepath.Join(integrationtests.OemInputDir, integrationtests.File1))
	time.Sleep(integrationtests.LessPauseTime)

	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		require.Eventually(t, func() bool {
			checkFile := filepath.Join(integrationtests.OemOutputDir, fmt.Sprintf("test-image-sim%d.tiff", i))
			return assert.FileExists(t, checkFile)
		}, integrationtests.PauseTime, time.Second)
	}
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

	// check output file status for multiple output files
	jobs, err := jobRepoClient.RetrieveAll(nil)
	require.NoError(t, err)
	assert.NotNil(t, jobs)
	for _, job := range jobs {
		assert.Equal(t, pkg.StatusComplete, job.Status)
		for _, file := range job.PipelineDetails.OutputFiles {
			assert.Equal(t, pkg.FileStatusComplete, file.Status)
			assert.Equal(t, pkg.CreateUserFacingError("", nil), file.ErrorDetails)
			assert.Equal(t, pkg.OwnerNone, file.Owner)
		}
	}

}

// Tests adding an attribute parsing structure to consul then parsing an input file
// based on said structure.
func TestAttributeParser(t *testing.T) {
	jobRepoClient := job_repo.NewClient(integrationtests.JobRepositoryUrl, integrationtests.HttpTimeout, nil)
	defer integrationtests.CleanJobs(t, jobRepoClient)

	taskObj := helpers.CreateTestTask(integrationtests.FileParsable, "only-file")
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	// Add parsing schema to consul
	testclient := integrationtests.NewTestClient()

	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf(integrationtests.ConsulAttributeParserUrl, "Test/"), nil)
	require.NoError(t, err)
	token := integrationtests.GetConsulACLToken(t)
	request.Header.Set(integrationtests.ConsulHeaderKey, fmt.Sprintf(integrationtests.ConsulTokenFmt, token))
	response, err := testclient.Do(request)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, err)

	request, err = http.NewRequest(http.MethodPut, fmt.Sprintf(integrationtests.ConsulAttributeParserUrl, path.Join("Test", "Id")), bytes.NewBuffer([]byte("t")))
	require.NoError(t, err)
	request.Header.Set(integrationtests.ConsulHeaderKey, fmt.Sprintf(integrationtests.ConsulTokenFmt, token))
	response, err = testclient.Do(request)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, err)

	request, err = http.NewRequest(http.MethodPut, fmt.Sprintf(integrationtests.ConsulAttributeParserUrl, path.Join("Test", "DataType")), bytes.NewBuffer([]byte("int")))
	require.NoError(t, err)
	request.Header.Set(integrationtests.ConsulHeaderKey, fmt.Sprintf(integrationtests.ConsulTokenFmt, token))
	response, err = testclient.Do(request)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, err)

	// Restart data organizer
	err = PipelineSimFactory.StopServices(integrationtests.ServiceDataOrg)
	require.NoError(t, err)
	err = PipelineSimFactory.StartServiceWithWait(integrationtests.ServiceDataOrg)
	require.NoError(t, err)
	// wait for container to start up
	time.Sleep(5 * time.Second)
	// Process file
	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File1), path.Join(integrationtests.OemInputDir, integrationtests.FileParsable))
	defer integrationtests.CleanFiles(integrationtests.FileParsable, []string{integrationtests.FileParsableOut})

	assert.FileExists(t, path.Join(integrationtests.OemInputDir, integrationtests.FileParsable))

	require.Eventually(t, func() bool {
		return assert.FileExists(t, path.Join(integrationtests.OemOutputDir, integrationtests.FileParsableOut))
	}, integrationtests.PauseTime, time.Second)

	// Check logs for attributes
	response, err = testclient.Get(integrationtests.GetJobUrl)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Test":"1"`)
}

// TestResultsMQTT tests that the pipelines generates a metadata result from a given job and publishes the result to mqtt.
func TestResultsMQTT(t *testing.T) {

	// create a new MQTT client
	opts := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1883").SetClientID("go-mqtt-int-test-client")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		require.NoError(t, token.Error())
	}

	messageChan := make(chan mqtt.Message)
	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		messageChan <- msg
	}

	// subscribe to a topic
	topic := "mqtt-export/#"
	qos := 0
	if token := client.Subscribe(topic, byte(qos), messageHandler); token.Wait() && token.Error() != nil {
		require.NoError(t, token.Error())
	}

	// Create new task with only-results
	taskObj := helpers.CreateTestTask(integrationtests.File4, "only-results")
	taskHttp := httpexpect.Default(t, integrationtests.TaskLauncherUrl)
	taskObj.Id = integrationtests.POST_Task(taskHttp, &taskObj, http.StatusCreated, false)
	defer integrationtests.DELETE_Task(taskHttp, &taskObj, http.StatusOK, false)

	// Create a job
	integrationtests.CopyFile(t, path.Join(integrationtests.LocalDir, integrationtests.File4), path.Join(integrationtests.OemInputDir, integrationtests.File4))
	defer integrationtests.CleanFiles(integrationtests.File4, []string{integrationtests.File4out})

	// wait for MQTT messages to arrive
	require.Eventually(t, func() bool {
		for {
			select {
			case msg := <-messageChan:
				assert.Equal(t, string(msg.Payload()), "filename:gateway=/tmp/files/input/test-image-4.tiff; results:CellCount, 598")
				return true
			}
		}
	}, integrationtests.PauseTime, time.Second)

}
