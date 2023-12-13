/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package functions

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	fileSenderMocks "aicsd/as-file-receiver-oem/clients/file_sender/mocks"
	"aicsd/pkg"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	appContext interfaces.AppFunctionContext
	lc         logger.MockLogger
)

const (
	oemFileHostname     = "oem"
	gatewayFileHostname = "gateway"
	testOutputFolder    = "test"
	file                = "test-image.tiff"
	file2               = "test-image2.tiff"
	fileWithSpaces      = "test image.tiff"
)

func createTestEvent(t *testing.T, job types.Job) dtos.Event {
	t.Helper()
	event := dtos.NewEvent(pkg.OwnerFileSenderGateway, pkg.OwnerFileSenderGateway, pkg.OwnerFileSenderGateway)
	event.AddObjectReading(pkg.ResourceNameJob, job)
	return event
}

func TestMain(m *testing.M) {
	correlationId := uuid.New().String()
	lc := logger.NewMockClient()
	appContext = appsdk.NewAppFuncContextForTest(correlationId, lc)

	os.Exit(m.Run())
}

func TestPipelineReceiver_ProcessEvent(t *testing.T) {
	testJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, gatewayFileHostname)

	tests := []struct {
		Name        string
		Input       interface{}
		ExpectedErr error
	}{
		{"happy path", createTestEvent(t, testJob), nil},
		{"no pipeline data", nil, pkg.ErrEmptyInput},
		{"invalid input type", dtos.Address{}, pkg.ErrInvalidInput},
		{"invalid event fields", dtos.Event{}, pkg.ErrInvalidInput},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			gotContinuePipeline, gotErr := testReceiver.ProcessEvent(appContext, test.Input)
			if test.ExpectedErr != nil {
				require.Error(t, gotErr.(error))
				assert.Contains(t, gotErr.(error).Error(), test.ExpectedErr.Error())
				assert.False(t, gotContinuePipeline)
				return
			}
			assert.Equal(t, gotErr, test.ExpectedErr)
			assert.True(t, gotContinuePipeline)
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
}

func TestPipelineReceiver_UpdateJobRepoOwner(t *testing.T) {
	_ = os.Mkdir(testOutputFolder, pkg.FolderPermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file), []byte{}, pkg.FilePermissions)
	testJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, gatewayFileHostname)
	invalidJob := testJob
	invalidJob.Id = "-1"
	testJobUpdated := testJob
	testJobUpdated.Owner = pkg.OwnerFileRecvOem

	tests := []struct {
		Name        string
		InputJob    types.Job
		UpdatedJob  types.Job
		UpdateErr   error
		ExpectedErr error
	}{
		{"happy path", testJob, testJobUpdated, nil, nil},
		{"job repo update err", invalidJob, types.Job{}, pkg.ErrUpdating, pkg.ErrUpdating},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			testReceiver.lc = lc
			jobFields := make(map[string]interface{})
			jobFields[types.JobOwner] = pkg.OwnerFileRecvOem
			testReceiver.job = test.InputJob
			repoMock.On("Update", testReceiver.job.Id, jobFields).Return(test.UpdatedJob, test.UpdateErr)
			gotContinuePipeline, gotErr := testReceiver.UpdateJobRepoOwner(appContext, nil)
			if test.ExpectedErr != nil {
				require.Error(t, gotErr.(error))
				assert.Contains(t, gotErr.(error).Error(), test.ExpectedErr.Error())
				assert.False(t, gotContinuePipeline)
				return
			}
			assert.Equal(t, gotErr, test.ExpectedErr)
			assert.Equal(t, test.UpdatedJob.Owner, testReceiver.job.Owner)
			assert.True(t, gotContinuePipeline)
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	_ = os.RemoveAll(testOutputFolder)
}

func TestPipelineReceiver_PullFilePositive(t *testing.T) {
	_ = os.Mkdir(testOutputFolder, pkg.FolderPermissions)
	testFileBytes := []byte{'t', 'e', 's', 't'}
	_ = os.WriteFile(filepath.Join(testOutputFolder, file), testFileBytes, pkg.FilePermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file2), testFileBytes, pkg.FilePermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, fileWithSpaces), testFileBytes, pkg.FilePermissions)
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	expectedJobMultiOutput := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	expectedJobMultiOutput.PipelineDetails.OutputFiles = outFiles
	testJobWithSpaces := testJob
	testJobWithSpaces.PipelineDetails.OutputFiles[0].Name = fileWithSpaces

	tests := []struct {
		Name  string
		Input types.Job
	}{
		{"happy path - first output file", testJob},
		{"happy path - second output file", expectedJobMultiOutput},
		{"happy path - one output file with spaces", testJobWithSpaces},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			testReceiver.lc = lc
			testReceiver.job = test.Input
			for k, _ := range test.Input.PipelineDetails.OutputFiles {
				senderMock.On("TransmitFile", test.Input.Id, strconv.FormatInt(int64(k), 10)).Return(testFileBytes, nil)
			}
			gotContinuePipeline, gotErr := testReceiver.PullFile(appContext, nil)
			assert.Nil(t, gotErr)
			assert.True(t, gotContinuePipeline)
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	_ = os.RemoveAll(testOutputFolder)
}

func TestPipelineReceiver_PullFileNegative(t *testing.T) {
	_ = os.Mkdir(testOutputFolder, pkg.FolderPermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file), []byte{}, pkg.FilePermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file2), []byte{}, pkg.FilePermissions)
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	testJobMultiOutput := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile("", pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultiOutput.PipelineDetails.OutputFiles = outFiles
	testJobWithSpaces := testJob
	testJobWithSpaces.PipelineDetails.OutputFiles[0].Name = strings.Replace(testJobWithSpaces.PipelineDetails.OutputFiles[0].Name, "out", "out ", 1)
	invalidJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	invalidJob.Id = "-1"
	invalidJob2 := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	invalidJob.Id = "-2"
	testJobBlankFileLocation := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	testJobBlankFileLocation.PipelineDetails.OutputFileHost = ""
	testJobBlankFileLocation.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile("", pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testFileBytes := []byte{'t', 'e', 's', 't'}
	transmissionFailedJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, gatewayFileHostname)
	transmissionFailedJob.Status = pkg.StatusFileError
	transmissionFailedJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting)

	tests := []struct {
		Name        string
		Input       types.Job
		FileBytes   []byte
		ClientErr   []error
		ExpectedErr error
	}{
		{"client err - empty file bytes", invalidJob, nil, []error{pkg.ErrFileTransmitting}, pkg.ErrFileTransmitting},
		{"job transmission failed status remains - invalid job", invalidJob2, testFileBytes, []error{pkg.ErrFileTransmitting}, pkg.ErrFileTransmitting},
		{"job transmission failed status remains - nil bytes", transmissionFailedJob, nil, []error{pkg.ErrFileTransmitting}, pkg.ErrFileTransmitting},
		{"invalid output file location", testJobBlankFileLocation, testFileBytes, []error{nil}, pkg.ErrFileWrite},
		{"invalid first file, valid second file", testJobMultiOutput, testFileBytes, []error{pkg.ErrFileTransmitting, nil}, pkg.ErrFileTransmitting},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			testReceiver.lc = lc
			testReceiver.job = test.Input
			for k, _ := range test.Input.PipelineDetails.OutputFiles {
				senderMock.On("TransmitFile", test.Input.Id, strconv.FormatInt(int64(k), 10)).Return(test.FileBytes, test.ClientErr[k])
			}
			gotContinuePipeline, gotErr := testReceiver.PullFile(appContext, nil)
			if gotErr == nil {
				t.Fatal("TestPipelineReceiver_PullFileNegative expects an error, but none was received")
			}
			require.Error(t, gotErr.(error))
			assert.Contains(t, gotErr.(error).Error(), test.ExpectedErr.Error())
			assert.True(t, gotContinuePipeline)
			assert.Equal(t, pkg.StatusFileError, testReceiver.job.Status)
			assert.Contains(t, testReceiver.job.ErrorDetails.Error, test.ExpectedErr.Error())

			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	_ = os.RemoveAll(testOutputFolder)
}

func TestPipelineReceiver_ArchiveFile(t *testing.T) {
	_ = os.Mkdir(testOutputFolder, pkg.FolderPermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file), []byte{}, pkg.FilePermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file2), []byte{}, pkg.FilePermissions)
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultiOutput := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultiOutput.PipelineDetails.OutputFiles = outFiles

	invalidJob := testJob
	invalidJob.Id = "-1"

	tests := []struct {
		Name             string
		InputJob         types.Job
		ContinuePipeline bool
		ExpectedErr      error
	}{
		{"happy path - single output file", testJob, true, nil},
		{"happy path - two output files", testJobMultiOutput, true, nil},
		{"failed to archive file", invalidJob, true, pkg.ErrFileArchiving},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			testReceiver.lc = lc
			testReceiver.job = test.InputJob
			senderMock.On("ArchiveFile", testReceiver.job.Id).Return(test.ExpectedErr)
			gotContinuePipeline, gotErr := testReceiver.ArchiveFile(appContext, nil)
			if test.ExpectedErr != nil {
				require.Error(t, gotErr.(error))
				assert.Contains(t, gotErr.(error).Error(), test.ExpectedErr.Error())
				return
			}
			assert.Equal(t, test.ExpectedErr, gotErr)
			assert.Equal(t, gotContinuePipeline, test.ContinuePipeline)
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	_ = os.RemoveAll(testOutputFolder)
}

func TestPipelineReceiver_UpdateJobRepoComplete(t *testing.T) {
	_ = os.Mkdir(testOutputFolder, pkg.FolderPermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file2), []byte{}, pkg.FilePermissions)
	_ = os.WriteFile(filepath.Join(testOutputFolder, file), []byte{}, pkg.FilePermissions)
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultiOutput := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultiOutput.PipelineDetails.OutputFiles = outFiles
	testJobMultiOutputUpdated := testJobMultiOutput
	testJobMultiOutputUpdated.Owner = pkg.OwnerNone
	testJobMultiOutputUpdated.Status = pkg.StatusComplete

	invalidJob := testJob
	invalidJob.Id = "-1"
	testJobUpdated := testJob
	testJobUpdated.Owner = pkg.OwnerNone
	testJobUpdated.Status = pkg.StatusComplete
	testJobUpdated.ErrorDetails = pkg.CreateUserFacingError("", nil)
	testJobUpdated.PipelineDetails.OutputFileHost = oemFileHostname
	transmissionFailedJob := testJob
	transmissionFailedJob.Status = pkg.StatusFileError
	transmissionFailedJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting)
	transmissionFailedJobUpdated := transmissionFailedJob

	jobFields := make(map[string]interface{})
	jobFields[types.JobOwner] = pkg.OwnerNone
	jobFields[types.JobStatus] = pkg.StatusComplete
	jobFields[types.JobPipelineOutputHost] = oemFileHostname
	jobFields[types.JobErrorDetailsOwner] = ""
	jobFields[types.JobErrorDetailsErrorMsg] = ""

	jobFields2 := make(map[string]interface{})
	jobFields2[types.JobOwner] = pkg.OwnerNone
	jobFields2[types.JobStatus] = pkg.StatusFileError
	jobFields2[types.JobErrorDetailsOwner] = pkg.OwnerFileRecvOem
	jobFields2[types.JobErrorDetailsErrorMsg] = pkg.ErrFileTransmitting.Error()
	jobFields2[types.JobPipelineOutputHost] = oemFileHostname

	jobFields3 := make(map[string]interface{})
	jobFields3[types.JobOwner] = pkg.OwnerNone
	jobFields3[types.JobStatus] = pkg.StatusComplete
	jobFields3[types.JobPipelineOutputHost] = oemFileHostname
	jobFields3[types.JobErrorDetailsOwner] = ""
	jobFields3[types.JobErrorDetailsErrorMsg] = ""

	tests := []struct {
		Name         string
		InputJob     types.Job
		UpdatedJob   types.Job
		UpdateFields map[string]interface{}
		UpdateErr    error
		ExpectedErr  error
	}{
		{"happy path - first output file", testJob, testJobUpdated, jobFields, nil, nil},
		{"happy path - second output file", testJobMultiOutput, testJobMultiOutputUpdated, jobFields3, nil, nil},
		{"job transmission failed status remains", transmissionFailedJob, transmissionFailedJobUpdated, jobFields2, nil, nil},
		{"job repo update err", invalidJob, types.Job{}, jobFields, pkg.ErrUpdating, pkg.ErrUpdating},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testReceiver := NewPipelineReceiver(&repoMock, &senderMock, oemFileHostname, testOutputFolder)
			testReceiver.lc = lc
			testReceiver.job = test.InputJob

			repoMock.On("Update", testReceiver.job.Id, test.UpdateFields).Return(test.UpdatedJob, test.UpdateErr)
			gotContinuePipeline, gotErr := testReceiver.UpdateJobRepoComplete(appContext, nil)
			assert.False(t, gotContinuePipeline)
			if test.ExpectedErr != nil {
				require.Error(t, gotErr.(error))
				assert.Contains(t, gotErr.(error).Error(), test.ExpectedErr.Error())
				assert.Equal(t, test.UpdatedJob.Status, testReceiver.job.Status)
				assert.Equal(t, test.UpdatedJob.ErrorDetails, testReceiver.job.ErrorDetails)
				return
			}
			assert.Equal(t, gotErr, test.ExpectedErr)
			assert.Equal(t, test.UpdatedJob.Owner, testReceiver.job.Owner)
			assert.Equal(t, test.UpdatedJob.Status, testReceiver.job.Status)
			assert.Equal(t, test.UpdatedJob.ErrorDetails, testReceiver.job.ErrorDetails)
			assert.Equal(t, test.UpdatedJob.PipelineDetails.OutputFileHost, testReceiver.job.PipelineDetails.OutputFileHost)
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	_ = os.RemoveAll(testOutputFolder)
}
