/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	taskLauncherMocks "aicsd/ms-data-organizer/clients/task_launcher/mocks"
	"aicsd/pkg"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
)

const (
	fileHostname = "gateway"
	file         = "outfile1.tiff"
	file2        = "outfile2.tiff"
	bogusFile    = "bogus.tiff"
	bogusDir     = "bogus"
	testDir      = "test"
)

var (
	appContext              interfaces.AppFunctionContext
	mockBackgroundPublisher *mocks.BackgroundPublisher
	mockAppService          *mocks.ApplicationService

	archiveFolder = filepath.Join(".", "test", "archive")
	rejectFolder  = filepath.Join(".", "test", "reject")
	testJob       = helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
)

func TestMain(m *testing.M) {
	mockBackgroundPublisher = &mocks.BackgroundPublisher{}
	mockAppService = &mocks.ApplicationService{}
	mockAppService.On("BuildContext", mock.Anything, common.ContentTypeJSON).Return(appContext)
	correlationId := uuid.New().String()
	lc := logger.NewMockClient()
	appContext = appsdk.NewAppFuncContextForTest(correlationId, lc)

	os.Exit(m.Run())
}

func TestFileSender_RetryOnStartup(t *testing.T) {
	tearDownTestResources := helpers.SetupTestFiles(t)
	defer tearDownTestResources(t)

	testJob.Status = pkg.StatusIncomplete
	invalidJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	invalidJob.PipelineDetails.OutputFileHost = ""
	testJobInvalidFile := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	testJobInvalidFile.PipelineDetails.OutputFiles[0].DirName = ""
	testJobInvalidFile.PipelineDetails.OutputFiles[0].Name = ""

	tests := []struct {
		Name        string
		Expected    *[]types.Job
		ExpectedErr error
	}{
		{"happy path", &[]types.Job{testJob}, nil},
		{"retrieve failed", &[]types.Job{testJob}, pkg.ErrRetrieving},
		{"invalid job", &[]types.Job{invalidJob}, pkg.ErrJobInvalid},
		{"unable to publish job", &[]types.Job{testJob}, pkg.ErrPublishing},
		{"unable to update job", &[]types.Job{testJob}, pkg.ErrUpdating},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileSenderGateway).Return(*test.Expected, test.ExpectedErr)
			mockAppService.On("publishJob", test.Expected, appContext).Return(test.ExpectedErr)
			mockBackgroundPublisher.On("Publish", mock.Anything, mock.Anything).Return(test.ExpectedErr)

			if test.ExpectedErr == pkg.ErrUpdating {
				repoMock.On("Update", mock.Anything, mock.Anything).Return(&[]types.Job{}, test.ExpectedErr)
			}

			for _, job := range *test.Expected {
				jobFields := make(map[string]interface{})
				jobFields[types.JobStatus] = job.Status
				jobFields[types.JobOwner] = job.Owner
				if job.ErrorDetails.Error != "" {
					jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
					jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				}
				repoMock.On("Update", job.Id, jobFields).Return(job, nil)
			}

			err = testController.RetryOnStartup()
			if test.ExpectedErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Contains(t, err.Error(), test.ExpectedErr.Error())
			}

			for _, job := range *test.Expected {
				for _, outFile := range job.PipelineDetails.OutputFiles {
					assert.Equal(t, pkg.FileStatusIncomplete, job.PipelineDetails.OutputFiles[0].Status)
					assert.Equal(t, pkg.OwnerFileSenderGateway, outFile.Owner)
					assert.Equal(t, pkg.CreateUserFacingError("", nil), job.ErrorDetails)
				}
			}
		})
	}
}

func TestFileSender_HandleNewJob(t *testing.T) {
	tearDownTestResources := helpers.SetupTestFiles(t)
	defer tearDownTestResources(t)

	testJob.Status = pkg.StatusIncomplete
	invalidJob := testJob
	invalidJob.PipelineDetails.OutputFileHost = ""
	invalidJob.Status = pkg.StatusFileError
	testJobInvalidFile := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	testJobInvalidFile.PipelineDetails.OutputFiles[0].DirName = ""
	testJobInvalidFile.PipelineDetails.OutputFiles[0].Name = ""
	testJobInvalidFile.Status = pkg.StatusFileError

	tests := []struct {
		Name                 string
		Expected             *types.Job
		ExpectedUpdateErr    error
		ExpectedPublisherErr error
		ExpectedStatusCode   int
		ExpectedErrorMsg     string
	}{
		{"happy path", &testJob, nil, nil, http.StatusOK, ""},
		{"bad request body", nil, nil, nil, http.StatusBadRequest, "failed to unmarshal request"},
		{"unable to publish job", &testJob, nil, pkg.ErrPublishing, http.StatusOK, pkg.ErrPublishing.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			if test.Expected != nil {
				jobFields := make(map[string]interface{})
				jobFields[types.JobOwner] = pkg.OwnerFileSenderGateway
				jobFields[types.JobStatus] = pkg.StatusIncomplete
				jobFields[types.JobPipelineOutputFiles] = test.Expected.PipelineDetails.OutputFiles
				if test.Expected.Status == pkg.StatusFileError {
					jobFields[types.JobStatus] = pkg.StatusFileError
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderGateway
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileInvalid.Error()
				}
				requestBody, err = json.Marshal(test.Expected)
				repoMock.On("Update", test.Expected.Id, jobFields).Return(*test.Expected, test.ExpectedUpdateErr)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}

			mockAppService.On("publishJob", test.Expected, appContext).Return(test.ExpectedPublisherErr)
			mockBackgroundPublisher.On("Publish", mock.Anything, mock.Anything).Return(test.ExpectedPublisherErr)

			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			testController.HandleNewJob(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			assert.NotNil(t, resp)
			assert.Equal(t, test.ExpectedStatusCode, resp.StatusCode)
			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			} else if test.Expected.Id != "" && resp.StatusCode == http.StatusOK {
				assert.Contains(t, testController.jobMap, test.Expected.Id)
			}
			if test.Expected.Status == pkg.StatusFileError {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErrorMsg)
			}
			repoMock.AssertExpectations(t)
		})
	}
}

func TestFileSender_TransmitFile(t *testing.T) {
	tearDownTestResources := helpers.SetupTestFiles(t)
	defer tearDownTestResources(t)

	expected := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	expectedJobMultiOutput := expected
	expectedJobMultiOutput.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	invalidJob := expected
	invalidJob.Id = "1000"
	invalidFileJob := expected
	invalidFileJob.PipelineDetails.OutputFileHost = "bogus"
	invalidFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(bogusFile, pkg.FileStatusInvalid, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil))}
	invalidFileJob.Status = pkg.StatusFileError

	tests := []struct {
		Name                string
		Expected            *types.Job
		FileId              int
		ExpectedKeyErr      error
		ExpectedValidJobErr error
		ExpectedReadFileErr error
		ExpectedStatusCode  int
	}{
		{"happy path - first output file", &expected, 0, nil, nil, nil, http.StatusOK},
		{"happy path - second output file", &expectedJobMultiOutput, 1, nil, nil, nil, http.StatusOK},
		{"job does not exist internally", &invalidJob, 0, pkg.ErrJobInvalid, nil, nil, http.StatusInternalServerError},
		{"invalid job", &invalidJob, 1, nil, nil, nil, http.StatusInternalServerError},
		{"unable to read job file - file does not exist", &invalidFileJob, 0, nil, nil, errors.New("failed"), http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			jobFields := make(map[string]interface{})
			jobFields[types.JobPipelineOutputFiles] = test.Expected.PipelineDetails.OutputFiles
			repoMock.On("Update", test.Expected.Id, jobFields).Return(*test.Expected, nil)

			// Add job to testController.JobMap if needed to test all err conditions
			if test.Expected != nil && test.Expected != &invalidJob {
				testController.jobMap[test.Expected.Id] = *test.Expected
			}

			req := httptest.NewRequest("GET", "http://localhost", nil)
			w := httptest.NewRecorder()
			req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Expected.Id, pkg.FileIdKey: strconv.FormatInt(int64(test.FileId), 10)})

			testController.TransmitFile(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// Rm job from testController.JobMap if needed
			if test.Expected != nil && test.Expected != &invalidJob {
				delete(testController.jobMap, test.Expected.Id)
			}

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK && test.ExpectedReadFileErr != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedReadFileErr.Error())
			}
		})
	}
}

func TestFileSender_ArchiveFilesPositive(t *testing.T) {
	tearDownTestResources := helpers.SetupTestFiles(t)
	defer tearDownTestResources(t)

	expected := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	expected.Status = pkg.StatusComplete
	expected.InputFile.ArchiveName = mock.Anything
	expectedJobMultiOutput := expected
	expectedJobMultiOutput.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}

	tests := []struct {
		Name               string
		Expected           types.Job
		ExpectedErr        error
		ExpectedStatusCode int
		ExpectedFiles      int
	}{
		{"happy path - one output file", expected, nil, http.StatusOK, 2},
		{"happy path - two output files", expectedJobMultiOutput, nil, http.StatusOK, 3},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "http://localhost", nil)
			w := httptest.NewRecorder()

			// Add job to testController.JobMap if needed to test all err conditions
			testController.jobMap[test.Expected.Id] = test.Expected
			req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Expected.Id})
			repoMock.On("Update", test.Expected.Id, mock.Anything).Return(test.Expected, nil)

			testController.ArchiveFile(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// Rm job from testController.JobMap if needed
			delete(testController.jobMap, test.Expected.Id)

			// clean up the directory and rename the files back
			files, err := os.ReadDir(archiveFolder)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedFiles, len(files))
			for _, f := range files {
				if !f.IsDir() {
					oldFile := strings.Join([]string{strings.Split(f.Name(), "_archive")[0], filepath.Ext(f.Name())}, "")
					t.Logf("here is oldFile %s", oldFile)

					err = os.Rename(filepath.Join(archiveFolder, f.Name()), filepath.Join(".", "test", oldFile))
					require.NoError(t, err)
				}
			}

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK && test.ExpectedErr != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErr.Error())
			}
		})
	}
}

func TestFileSender_ArchiveFilesNegative(t *testing.T) {
	tearDownTestResources := helpers.SetupTestFiles(t)
	defer tearDownTestResources(t)

	badInputFileJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	badInputFileJob.InputFile.DirName = bogusDir
	badInputFileJob.InputFile.Name = bogusFile
	badInputFileJob.Status = pkg.StatusFileError
	badInputFileJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileArchiving)
	badOutputFileJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	badOutputFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile("", pkg.FileStatusInvalid, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil))}
	badOutputFileJob.PipelineDetails.OutputFiles[0].DirName = ""
	badOutputFileJob.Status = pkg.StatusFileError
	badOutputFileJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileInvalid)

	tests := []struct {
		Name               string
		Expected           *types.Job
		ExpectedErr        error
		ExpectedStatusCode int
		ExpectedFiles      int
	}{
		{"invalid job id", nil, errors.New("missing jobid in url"), http.StatusBadRequest, 0},
		{"bad input file", &badInputFileJob, errors.New("failed to archive input file"), http.StatusInternalServerError, 0},
		// Note: the below test case responds with a 200 as it tries to archive the output files that it can.
		// The error feedback for bad output files are captured in the job/file err details fields.
		{"bad output file", &badOutputFileJob, pkg.ErrFileInvalid, http.StatusOK, 1},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "http://localhost", nil)
			w := httptest.NewRecorder()

			jobFields := make(map[string]interface{})
			// Add job to testController.JobMap if needed to test all err conditions
			if test.Expected != nil {
				testController.jobMap[test.Expected.Id] = *test.Expected
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Expected.Id})

				if test.Expected.PipelineDetails.OutputFiles[0].DirName == "" {
					jobFields[types.JobPipelineOutputFiles] = test.Expected.PipelineDetails.OutputFiles
				}
				repoMock.On("Update", test.Expected.Id, mock.Anything).Return(*test.Expected, nil)
			}

			testController.ArchiveFile(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// Rm job from testController.JobMap if needed
			if test.Expected != nil {
				delete(testController.jobMap, test.Expected.Id)
			}

			// clean up the directory and rename the files back
			files, err := os.ReadDir(archiveFolder)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedFiles, len(files))
			for _, f := range files {
				if !f.IsDir() {
					oldFile := strings.Join([]string{strings.Split(f.Name(), "_archive")[0], filepath.Ext(f.Name())}, "")
					t.Logf("here is oldFile %s", oldFile)
					err = os.Rename(filepath.Join(archiveFolder, f.Name()), filepath.Join(".", "test", oldFile))
					require.NoError(t, err)
				}
			}

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK && test.ExpectedErr != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErr.Error())
			}
		})
	}
}

func TestFileSender_ImageConversionPositive(t *testing.T) {

	// Setup for non-zerobyte files
	const (
		testFile    = "test-image_archive.tiff"
		outputFile1 = "output1_archive.tiff"
		outputFile2 = "output2_archive.tiff"
	)
	require.NoError(t, os.Mkdir(filepath.Join(".", testDir), pkg.FolderPermissions))
	require.NoError(t, os.Mkdir(filepath.Join(archiveFolder), pkg.FolderPermissions))
	fb, err := os.ReadFile("../../integration-tests/sample-files/example.tiff")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, testFile), fb, pkg.FilePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, outputFile1), fb, pkg.FilePermissions))

	expected := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	expected.Status = pkg.StatusComplete
	expected.InputFile.ArchiveName = filepath.Join(".", archiveFolder, testFile)
	expected.PipelineDetails.OutputFiles[0].ArchiveName = filepath.Join(".", archiveFolder, outputFile1)
	expectedJobMultiOutput := expected
	expectedJobMultiOutput.InputFile.ArchiveName = filepath.Join(".", archiveFolder, testFile)
	expectedJobMultiOutput.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	expectedJobMultiOutput.PipelineDetails.OutputFiles[0].ArchiveName = filepath.Join(".", archiveFolder, outputFile1)
	expectedJobMultiOutput.PipelineDetails.OutputFiles[1].ArchiveName = filepath.Join(".", archiveFolder, outputFile2)

	tests := []struct {
		Name               string
		Expected           types.Job
		ExpectedErr        error
		ExpectedStatusCode int
		ExpectedFiles      int
	}{
		{"happy path - one output file", expected, nil, http.StatusOK, 4},
		{"happy path - multi output files", expectedJobMultiOutput, nil, http.StatusOK, 6},
	}

	for _, test := range tests {
		if strings.Contains(test.Name, "multi") {
			require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, outputFile2), fb, pkg.FilePermissions))
		}
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			repoMock.On("Update", test.Expected.Id, mock.Anything).Return(test.Expected, nil)

			testController.Convert(w, test.Expected, test.Expected.InputFile.ArchiveName)
			resp := w.Result()
			defer resp.Body.Close()

			// clean up the directory and rename the files back

			files, err := os.ReadDir(archiveFolder)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedFiles, len(files))

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK && test.ExpectedErr != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErr.Error())
			}
		})
	}
	// test file clean up
	t.Logf("tearing down test files for test case named: %s", t.Name())
	require.NoError(t, os.RemoveAll(filepath.Join(".", testDir)))
	require.NoError(t, os.RemoveAll(archiveFolder))
}

func TestFileSender_ImageConversionNegative(t *testing.T) {
	// Setup for negative tests
	const (
		testFile   = "test-image_archive.tiff"
		badInput   = "bad_archive.tiff"
		badOutput1 = "out_bad1_archive.tiff"
		badOutput2 = "out_bad2_archive.tiff"
	)
	require.NoError(t, os.Mkdir(filepath.Join(".", testDir), pkg.FolderPermissions))
	require.NoError(t, os.Mkdir(filepath.Join(archiveFolder), pkg.FolderPermissions))

	require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, badInput), []byte{0100}, pkg.FilePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, badOutput1), []byte{0010}, pkg.FilePermissions))

	badInputFileJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	badInputFileJob.InputFile.Name = badInput
	badInputFileJob.InputFile.ArchiveName = filepath.Join(".", archiveFolder, badInput)

	badOutputFileJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	badOutputFileJob.InputFile.ArchiveName = filepath.Join(".", archiveFolder, testFile)
	badOutputFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(badOutput1, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil))}
	badOutputFileJob.PipelineDetails.OutputFiles[0].ArchiveName = filepath.Join(".", archiveFolder, badOutput1)

	badMultiOutputFileJob := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, fileHostname)
	badMultiOutputFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(badOutput1, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(badOutput2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	badMultiOutputFileJob.PipelineDetails.OutputFiles[0].ArchiveName = filepath.Join(".", archiveFolder, badOutput1)
	badMultiOutputFileJob.PipelineDetails.OutputFiles[1].ArchiveName = filepath.Join(".", archiveFolder, badOutput2)
	tests := []struct {
		Name               string
		Expected           types.Job
		ExpectedErr        error
		ExpectedStatusCode int
		ExpectedFiles      int
	}{
		{"negative - bad input", badInputFileJob, errors.New("failed to open archival file for conversion"), http.StatusInternalServerError, 2},
		{"negative - bad output", badOutputFileJob, errors.New("failed to open archival output file for conversion"), http.StatusInternalServerError, 3},
		{"negative - bad multi output", badMultiOutputFileJob, errors.New("image: unknown format"), http.StatusInternalServerError, 4},
	}

	for _, test := range tests {
		if strings.Contains(test.Name, "output") {
			fb, err := os.ReadFile("../../integration-tests/sample-files/example.tiff")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, testFile), fb, pkg.FilePermissions))
		}
		if strings.Contains(test.Name, "multi") {
			require.NoError(t, os.WriteFile(filepath.Join(".", archiveFolder, badOutput2), []byte{0001}, pkg.FilePermissions))
		}
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(make([]types.Job, 0), nil)
			launcherMock := taskLauncherMocks.Client{}
			testController, err := New(logger.NewMockClient(), &repoMock, &launcherMock, mockBackgroundPublisher, mockAppService, fileHostname, archiveFolder, rejectFolder)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			repoMock.On("Update", test.Expected.Id, mock.Anything).Return(test.Expected, nil)

			testController.Convert(w, test.Expected, test.Expected.InputFile.ArchiveName)
			resp := w.Result()
			defer resp.Body.Close()

			// clean up the directory and rename the files back

			files, err := os.ReadDir(archiveFolder)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedFiles, len(files))

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK && test.ExpectedErr != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErr.Error())
			}
		})
		if strings.Contains(test.Name, "bad input") {
			require.NoError(t, os.Remove(test.Expected.InputFile.ArchiveName))
		}
	}
	// test file clean up
	t.Logf("tearing down test files for test case named: %s", t.Name())
	require.NoError(t, os.RemoveAll(filepath.Join(".", testDir)))
	require.NoError(t, os.RemoveAll(archiveFolder))
}
