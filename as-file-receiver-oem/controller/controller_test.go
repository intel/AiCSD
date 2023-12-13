/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	fileSenderMocks "aicsd/as-file-receiver-oem/clients/file_sender/mocks"
	"aicsd/pkg"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	oemFileHostname = "oem"
	testDir         = "test"
	testExt         = ".tiff"
)

var (
	file              = filepath.Join(testDir, "outfile1.tiff")
	file2             = filepath.Join(testDir, "outfile2.tiff")
	file3             = filepath.Join(testDir, "outfile3.tiff")
	file4             = filepath.Join(testDir, "outfile4.tiff")
	file5             = filepath.Join(testDir, "outfile5.tiff")
	file6             = filepath.Join(testDir, "outfile6.tiff")
	invalidFile       = ""
	dependentServices = wait.Services{wait.ServiceConsul, wait.ServiceRedis, wait.ServiceJobRepo}
)

func TestRetryOneJobPositive(t *testing.T) {
	testFileBytes := []byte{'t', 'e', 's', 't'}
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultFiles := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles.PipelineDetails.OutputFiles = outFiles
	testJobMultFiles.Id = "2"
	testJobMultFiles2 := testJobMultFiles
	outFiles2 := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file4, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file5, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles2.PipelineDetails.OutputFiles = outFiles2
	testJobMultFiles2.Id = "3"
	testJob2 := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJob2.Id = "1"
	testJob2.PipelineDetails.OutputFiles = outFiles
	testJobUpdated := testJob
	testJobUpdated.Owner = pkg.OwnerNone
	testJobUpdated.Status = pkg.StatusComplete
	testJobUpdated.ErrorDetails = pkg.CreateUserFacingError("", nil)
	testJobUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobUpdated.PipelineDetails.OutputFileHost = oemFileHostname
	testJobMultFilesUpdated := testJobUpdated
	testJobMultFilesUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated.Id = "2"
	testJobMultFilesUpdated2 := testJobMultFilesUpdated
	testJobMultFilesUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file3, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated2.Id = "3"
	testJobUpdated2 := testJobUpdated
	testJobUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}

	// make test directory specifically for writing output files
	_ = os.Mkdir(filepath.Join(".", testDir), 0777)

	tests := []struct {
		Name        string
		InputJob    []types.Job
		UpdatedJob  []types.Job
		RetrieveErr error
		TransmitErr []error
		UpdateErr   error
		ArchiveErr  error
		JobErr      error
	}{
		{"happy path - one job one file", []types.Job{testJob}, []types.Job{testJobUpdated}, nil, []error{nil}, nil, nil, nil},
		{"happy path - one job multiple files", []types.Job{testJobMultFiles}, []types.Job{testJobMultFilesUpdated}, nil, []error{nil, nil}, nil, nil, nil},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testController := New(logger.MockLogger{}, &repoMock, &senderMock, oemFileHostname, testDir, dependentServices)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(test.InputJob, test.RetrieveErr)
			jobFields := make(map[string]interface{})
			senderMock.On("ArchiveFile", test.InputJob[0].Id).Return(test.ArchiveErr)
			senderMock.On("Retry").Return(nil)
			repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[0], test.UpdateErr)

			for i, job := range test.UpdatedJob {
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusComplete
				jobFields[types.JobPipelineOutputHost] = oemFileHostname
				jobFields[types.JobPipelineOutputFiles] = job.PipelineDetails.OutputFiles
				senderMock.On("TransmitFile", test.InputJob[i].Id, mock.Anything).Return(testFileBytes, nil)
			}
			err := testController.RetryOnStartup()
			if test.JobErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.JobErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, err, test.JobErr)
			}
			for i, job := range test.UpdatedJob {
				for fileId, outfile := range job.PipelineDetails.OutputFiles {
					assert.Equal(t, testDir, outfile.DirName)
					assert.Equal(t, testExt, outfile.Extension)
					if test.JobErr != nil {
						if test.TransmitErr[i] != nil {
							assert.Equal(t, pkg.FileStatusTransmissionFailed, outfile.Status)
							assert.Equal(t, pkg.OwnerFileRecvOem, outfile.Owner)
						}
					}
					// TODO try to simplify test checking logic
					if test.JobErr != nil || job.Status == pkg.StatusFileError {
						// TODO check file status error
					} else {
						assert.Equal(t, testDir, outfile.DirName)
						assert.Equal(t, filepath.Join("test", fmt.Sprintf("outfile%d.tiff", fileId+1)), outfile.Name)
						assert.Equal(t, testExt, outfile.Extension)
						assert.Equal(t, pkg.FileStatusComplete, outfile.Status)
						assert.Equal(t, pkg.CreateUserFacingError("", nil), outfile.ErrorDetails)
						assert.Equal(t, pkg.OwnerNone, outfile.Owner)
						assert.Equal(t, pkg.OwnerNone, job.Owner)
					}
				}
			}
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	// test file clean up
	_ = os.RemoveAll(testDir)
}

func TestRetryOneJobNegative(t *testing.T) {
	testFileBytes := []byte{'t', 'e', 's', 't'}
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultFiles := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles.PipelineDetails.OutputFiles = outFiles
	testJobMultFiles.Id = "2"
	testJobInvalidFileLocation := testJob
	testJobInvalidFileLocation.PipelineDetails.OutputFileHost = ""
	testJobInvalidFileLocation.PipelineDetails.OutputFiles = []types.OutputFile{}
	transmissionFailedJob := testJob
	transmissionFailedJob.Status = pkg.StatusFileError
	transmissionFailedJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(filepath.Join(".", "bogus"), pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	transmissionFailedJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting)
	transmissionFailedJobUpdated := transmissionFailedJob
	transmissionFailedJobUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(filepath.Join(".", "bogus"), pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	transmissionFailedJobUpdated.Status = pkg.StatusFileError

	testJobOneFileInvalid := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobOneFileInvalid.Status = pkg.StatusFileError
	testJobOneFileInvalid.PipelineDetails.OutputFiles = []types.OutputFile{
		helpers.CreateTestFile(invalidFile, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobOneFileInvalidUpdated := testJobOneFileInvalid
	testJobInvalidFileLocation.PipelineDetails.OutputFileHost = oemFileHostname
	testJobOneFileInvalidUpdated.PipelineDetails.OutputFiles = []types.OutputFile{
		helpers.CreateTestFile(invalidFile, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting))}
	testJobOneFileInvalidUpdated.Owner = pkg.OwnerFileRecvOem
	testJobOneFileInvalidUpdated.Status = pkg.StatusFileError

	testJobMultiFilesOneFileInvalid := testJobMultFiles
	testJobMultiFilesOneFileInvalid.Status = pkg.StatusIncomplete
	testJobMultiFilesOneFileInvalid.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(invalidFile, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultiFilesOneFileInvalidUpdated := testJobMultiFilesOneFileInvalid
	testJobMultiFilesOneFileInvalidUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(invalidFile, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting))}
	testJobMultiFilesOneFileInvalidUpdated.Owner = pkg.OwnerNone
	testJobMultiFilesOneFileInvalidUpdated.Status = pkg.StatusFileError

	// make test directory specifically for writing output files
	_ = os.Mkdir(filepath.Join(".", testDir), 0777)

	tests := []struct {
		Name        string
		InputJob    []types.Job
		UpdatedJob  []types.Job
		RetrieveErr error
		TransmitErr []error
		UpdateErr   error
		ArchiveErr  error
		JobErr      error
		RetryErr    error
	}{
		{"one job one file - invalid file", []types.Job{testJobOneFileInvalid}, []types.Job{testJobOneFileInvalidUpdated}, nil, []error{pkg.ErrFileTransmitting}, nil, nil, pkg.ErrFileTransmitting, nil},
		{"one job multiple files - invalid second file", []types.Job{testJobMultiFilesOneFileInvalid}, []types.Job{testJobMultiFilesOneFileInvalidUpdated}, nil, []error{nil, pkg.ErrFileTransmitting}, nil, nil, errors.New("failed"), nil},
		{"retrieve failed", []types.Job{}, []types.Job{}, pkg.ErrRetrieving, nil, nil, nil, pkg.ErrRetrieving, nil},
		{"failed to write file - invalid location", []types.Job{testJobInvalidFileLocation}, []types.Job{testJobInvalidFileLocation}, nil, []error{nil}, nil, nil, nil, nil},
		{"failed to retry", []types.Job{}, []types.Job{}, nil, nil, nil, nil, errors.New("failed to Retry"), errors.New("failed to Retry")},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testController := New(logger.NewMockClient(), &repoMock, &senderMock, oemFileHostname, testDir, dependentServices)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(test.InputJob, test.RetrieveErr)
			jobFields := make(map[string]interface{})
			if test.RetrieveErr != nil {
				return
			}

			for i, job := range test.UpdatedJob {
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusComplete
				jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
				jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				jobFields[types.JobPipelineOutputHost] = oemFileHostname
				jobFields[types.JobPipelineOutputFiles] = job.PipelineDetails.OutputFiles

				for j, _ := range job.PipelineDetails.OutputFiles {
					if test.TransmitErr[i] != nil {
						testController.lc.Debugf("nil file bytes from tests")
						senderMock.On("TransmitFile", test.InputJob[i].Id, strconv.FormatInt(int64(j), 10)).Return(nil, test.TransmitErr[j])
					} else {
						senderMock.On("TransmitFile", test.InputJob[i].Id, mock.Anything).Return(testFileBytes, test.TransmitErr[j])
					}
				}
				senderMock.On("ArchiveFile", job.Id).Return(test.ArchiveErr)
				repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[i], test.UpdateErr)
			}

			senderMock.On("Retry").Return(test.RetryErr)

			err := testController.RetryOnStartup()
			if test.JobErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.JobErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, err, test.JobErr)
			}
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	// test file clean up
	_ = os.RemoveAll(testDir)
}

func TestRetryOneJobNegative_InvalidJobId(t *testing.T) {
	testFileBytes := []byte{'t', 'e', 's', 't'}
	invalidJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	invalidJob.Id = "-1"
	var invalidJobCopy types.Job
	copier.CopyWithOption(&invalidJobCopy, &invalidJob, copier.Option{DeepCopy: true})

	// make test directory specifically for writing output files
	_ = os.Mkdir(filepath.Join(".", testDir), 0777)

	tests := []struct {
		Name        string
		InputJob    []types.Job
		UpdatedJob  []types.Job
		RetrieveErr error
		TransmitErr []error
		UpdateErr   error
		ArchiveErr  error
		JobErr      error
		RetryErr    error
	}{
		{"failed job repo update", []types.Job{invalidJob}, []types.Job{invalidJob}, nil, []error{nil}, pkg.ErrUpdating, nil, pkg.ErrUpdating, nil},
		{"failed to archive file", []types.Job{invalidJob}, []types.Job{invalidJob}, nil, []error{nil}, nil, pkg.ErrFileArchiving, errors.New("failed"), nil},                           // JobErr nil here bc we check file status before archiving and only archive valid status files
		{"failed file sender client - invalid job id", []types.Job{invalidJob}, []types.Job{invalidJob}, nil, []error{pkg.ErrFileTransmitting}, nil, nil, pkg.ErrFileTransmitting, nil}, // TODO: this tests for invalid job ID. We should have job repo validate for valid job IDs
		{"failed to Retry", []types.Job{invalidJob}, []types.Job{invalidJob}, nil, []error{nil}, nil, nil, errors.New("failed"), errors.New("failed to Retry")},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testController := New(logger.NewMockClient(), &repoMock, &senderMock, oemFileHostname, testDir, dependentServices)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(test.InputJob, test.RetrieveErr)
			jobFields := make(map[string]interface{})
			if test.RetrieveErr != nil {
				return
			}

			for i, job := range test.UpdatedJob {
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusComplete
				jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
				jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				jobFields[types.JobPipelineOutputHost] = oemFileHostname
				jobFields[types.JobPipelineOutputFiles] = job.PipelineDetails.OutputFiles

				for j, _ := range job.PipelineDetails.OutputFiles {
					if test.TransmitErr[i] != nil {
						testController.lc.Debugf("nil file bytes from tests")
						senderMock.On("TransmitFile", test.InputJob[i].Id, strconv.FormatInt(int64(j), 10)).Return(nil, test.TransmitErr[j])
					} else {
						senderMock.On("TransmitFile", test.InputJob[i].Id, mock.Anything).Return(testFileBytes, test.TransmitErr[j])
					}
				}

				senderMock.On("ArchiveFile", job.Id).Return(test.ArchiveErr)
				repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[i], test.UpdateErr)
			}

			senderMock.On("Retry").Return(test.RetryErr)

			err := testController.RetryOnStartup()
			if test.JobErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.JobErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, err, test.JobErr)
			}
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
			copier.CopyWithOption(&invalidJob, &invalidJobCopy, copier.Option{DeepCopy: true})
		})
	}
	// test file clean up
	_ = os.RemoveAll(testDir)
}

func TestRetryMultiJobPositive(t *testing.T) {
	testFileBytes := []byte{'t', 'e', 's', 't'}
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultFiles := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles.PipelineDetails.OutputFiles = outFiles
	testJobMultFiles.Id = "2"
	testJobMultFiles2 := testJobMultFiles
	outFiles2 := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file4, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file5, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles2.PipelineDetails.OutputFiles = outFiles2
	testJobMultFiles2.Id = "3"
	testJob2 := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJob2.Id = "1"
	testJob2.PipelineDetails.OutputFiles = outFiles

	testJobUpdated := testJob
	testJobUpdated.Owner = pkg.OwnerNone
	testJobUpdated.Status = pkg.StatusComplete
	testJobUpdated.ErrorDetails = pkg.CreateUserFacingError("", nil)
	testJobUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobUpdated.PipelineDetails.OutputFileHost = oemFileHostname
	testJobMultFilesUpdated := testJobUpdated
	testJobMultFilesUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated.Id = "2"
	testJobMultFilesUpdated2 := testJobMultFilesUpdated
	testJobMultFilesUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file3, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated2.Id = "3"
	testJobUpdated2 := testJobUpdated
	testJobUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}

	// make test directory specifically for writing output files
	_ = os.Mkdir(filepath.Join(".", testDir), 0777)

	tests := []struct {
		Name        string
		InputJob    []types.Job
		UpdatedJob  []types.Job
		RetrieveErr error
		TransmitErr []error
		UpdateErr   error
		ArchiveErr  error
		JobErr      error
	}{
		{"happy path - multiple jobs one file", []types.Job{testJob, testJob2}, []types.Job{testJobUpdated, testJobUpdated2}, nil, []error{nil, nil}, nil, nil, nil},
		{"happy path - multiple jobs multiple files", []types.Job{testJobMultFiles, testJobMultFiles2}, []types.Job{testJobMultFilesUpdated, testJobMultFilesUpdated2}, nil, []error{nil, nil, nil}, nil, nil, nil},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testController := New(logger.NewMockClient(), &repoMock, &senderMock, oemFileHostname, testDir, dependentServices)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(test.InputJob, test.RetrieveErr)
			jobFields := make(map[string]interface{})

			for i, job := range test.UpdatedJob {
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusComplete
				jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
				jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				jobFields[types.JobPipelineOutputHost] = oemFileHostname
				jobFields[types.JobPipelineOutputFiles] = job.PipelineDetails.OutputFiles

				for j, _ := range job.PipelineDetails.OutputFiles {
					senderMock.On("TransmitFile", test.InputJob[i].Id, mock.Anything).Return(testFileBytes, test.TransmitErr[j])
				}
				senderMock.On("ArchiveFile", job.Id).Return(test.ArchiveErr)
				repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[i], test.UpdateErr)
			}
			senderMock.On("Retry").Return(nil)
			err := testController.RetryOnStartup()
			if test.JobErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.JobErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, err, test.JobErr)
			}
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	// test file clean up
	_ = os.RemoveAll(testDir)
}

func TestRetryMultiJobNegative(t *testing.T) {
	testFileBytes := []byte{'t', 'e', 's', 't'}
	testJob := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJobMultFiles := testJob
	outFiles := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles.PipelineDetails.OutputFiles = outFiles
	testJobMultFiles.Id = "2"
	testJobMultFiles2 := testJobMultFiles
	outFiles2 := []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file4, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file5, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultFiles2.PipelineDetails.OutputFiles = outFiles2
	testJobMultFiles2.Id = "3"
	testJob2 := helpers.CreateTestJob(pkg.OwnerFileRecvOem, oemFileHostname)
	testJob2.Id = "1"
	testJob2.PipelineDetails.OutputFiles = outFiles
	invalidJob := testJob
	invalidJob.Id = "-1"
	testJobInvalidFileLocation := testJob
	testJobInvalidFileLocation.PipelineDetails.OutputFileHost = ""
	testJobInvalidFileLocation.PipelineDetails.OutputFiles = []types.OutputFile{}
	testJobUpdated := testJob
	testJobUpdated.Owner = pkg.OwnerNone
	testJobUpdated.Status = pkg.StatusComplete
	testJobUpdated.ErrorDetails = pkg.CreateUserFacingError("", nil)
	testJobUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobUpdated.PipelineDetails.OutputFileHost = oemFileHostname
	testJobMultFilesUpdated := testJobUpdated
	testJobMultFilesUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated.Id = "2"
	testJobMultFilesUpdated2 := testJobMultFilesUpdated
	testJobMultFilesUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file3, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	testJobMultFilesUpdated2.Id = "3"
	testJobUpdated2 := testJobUpdated
	testJobUpdated2.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil))}
	transmissionFailedJob := testJob
	transmissionFailedJob.Status = pkg.StatusFileError
	transmissionFailedJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(filepath.Join(".", "bogus"), pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}

	transmissionFailedJob.ErrorDetails = pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting)
	transmissionFailedJobUpdated := transmissionFailedJob
	transmissionFailedJobUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(filepath.Join(".", "bogus"), pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	transmissionFailedJobUpdated.Status = pkg.StatusFileError

	testJobOneFileInvalid := testJobMultFiles
	testJobOneFileInvalid.Status = pkg.StatusFileError
	testJobOneFileInvalid.PipelineDetails.OutputFiles = []types.OutputFile{
		helpers.CreateTestFile(invalidFile, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobOneFileInvalidUpdated := testJobOneFileInvalid
	testJobInvalidFileLocation.PipelineDetails.OutputFileHost = oemFileHostname
	testJobOneFileInvalidUpdated.PipelineDetails.OutputFiles = []types.OutputFile{ //helpers.CreateTestFile(file, pkg.FileStatusComplete, "", pkg.OwnerNone),
		helpers.CreateTestFile(invalidFile, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting))}
	testJobOneFileInvalidUpdated.Owner = pkg.OwnerFileRecvOem
	testJobOneFileInvalidUpdated.Status = pkg.StatusFileError

	testJobMultiFilesOneFileInvalid := testJobMultFiles
	testJobMultiFilesOneFileInvalid.Status = pkg.StatusIncomplete
	testJobMultiFilesOneFileInvalid.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(invalidFile, pkg.FileStatusIncomplete, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError("", nil))}
	testJobMultiFilesOneFileInvalidUpdated := testJobMultiFilesOneFileInvalid
	testJobMultiFilesOneFileInvalidUpdated.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusComplete, pkg.OwnerNone, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(invalidFile, pkg.FileStatusTransmissionFailed, pkg.OwnerFileRecvOem, pkg.CreateUserFacingError(pkg.OwnerFileRecvOem, pkg.ErrFileTransmitting))}
	testJobMultiFilesOneFileInvalidUpdated.Owner = pkg.OwnerNone
	testJobMultiFilesOneFileInvalidUpdated.Status = pkg.StatusFileError

	// make test directory specifically for writing output files
	_ = os.Mkdir(filepath.Join(".", testDir), 0777)

	tests := []struct {
		Name        string
		InputJob    []types.Job
		UpdatedJob  []types.Job
		RetrieveErr error
		TransmitErr []error
		UpdateErr   error
		ArchiveErr  error
		JobErr      error
		RetryErr    error
	}{
		{"one job multiple files - invalid second file", []types.Job{testJobMultiFilesOneFileInvalid}, []types.Job{testJobMultiFilesOneFileInvalidUpdated}, nil, []error{nil, pkg.ErrFileTransmitting}, nil, nil, errors.New("failed"), nil},
		{"multiple jobs multiple files - invalid second job second file", []types.Job{testJob, testJobMultiFilesOneFileInvalid}, []types.Job{testJobUpdated, testJobMultiFilesOneFileInvalidUpdated}, nil, []error{nil, nil, pkg.ErrFileTransmitting}, nil, nil, nil, nil},
		{"one job multiple files - failed to retry", []types.Job{testJobMultiFilesOneFileInvalid}, []types.Job{testJobMultiFilesOneFileInvalidUpdated}, nil, []error{nil, pkg.ErrFileTransmitting}, nil, nil, errors.New("failed"), errors.New("failed to Retry")},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			testController := New(logger.MockLogger{}, &repoMock, &senderMock, oemFileHostname, testDir, dependentServices)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvOem).Return(test.InputJob, test.RetrieveErr)
			jobFields := make(map[string]interface{})

			if test.RetrieveErr != nil {
				return
			}

			if test.InputJob[0].Id == "-1" {
				if test.JobErr == pkg.ErrFileArchiving {
					senderMock.On("TransmitFile", test.InputJob[0].Id, strconv.FormatInt(int64(0), 10)).Return(testFileBytes, test.TransmitErr[0])
				} else {
					senderMock.On("TransmitFile", mock.Anything, mock.Anything).Return(nil, test.TransmitErr[0])
				}
				senderMock.On("ArchiveFile", test.InputJob[0].Id).Return(test.ArchiveErr)
				repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[0], test.UpdateErr)
			}

			for i, job := range test.UpdatedJob {
				jobFields[types.JobOwner] = pkg.OwnerNone
				jobFields[types.JobStatus] = pkg.StatusComplete
				jobFields[types.JobErrorDetailsOwner] = job.ErrorDetails.Owner
				jobFields[types.JobErrorDetailsErrorMsg] = job.ErrorDetails.Error
				jobFields[types.JobPipelineOutputHost] = oemFileHostname
				jobFields[types.JobPipelineOutputFiles] = job.PipelineDetails.OutputFiles

				for j, _ := range job.PipelineDetails.OutputFiles {
					if test.TransmitErr[i] != nil {
						testController.lc.Debugf("nil file bytes from tests")
						senderMock.On("TransmitFile", test.InputJob[i].Id, strconv.FormatInt(int64(j), 10)).Return(nil, test.TransmitErr[j])
					} else {
						senderMock.On("TransmitFile", test.InputJob[i].Id, mock.Anything).Return(testFileBytes, test.TransmitErr[j])
					}
				}

				if test.InputJob[0].Id != "-1" {
					senderMock.On("ArchiveFile", job.Id).Return(test.ArchiveErr)
					repoMock.On("Update", mock.Anything, mock.Anything).Return(test.UpdatedJob[i], test.UpdateErr)
				}
			}
			senderMock.On("Retry").Return(test.RetryErr)
			err := testController.RetryOnStartup()
			if test.JobErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.JobErr.Error())
				//return
			} else {
				require.NoError(t, err)
				assert.Equal(t, err, test.JobErr)
			}
			for i, job := range test.UpdatedJob {
				for fileId, outfile := range job.PipelineDetails.OutputFiles {
					assert.Equal(t, testDir, outfile.DirName)
					assert.Equal(t, testExt, outfile.Extension)
					if test.JobErr != nil {
						if test.TransmitErr[i] != nil {
							assert.Equal(t, pkg.FileStatusTransmissionFailed, outfile.Status)
							assert.Equal(t, pkg.OwnerFileRecvOem, outfile.Owner)
						}
					}
					// TODO try to simplify test checking logic
					if test.JobErr != nil || job.Status == pkg.StatusFileError {
						// TODO check file status error
					} else {
						assert.Equal(t, testDir, outfile.DirName)
						assert.Equal(t, filepath.Join("test", fmt.Sprintf("outfile%d.tiff", fileId+1)), outfile.Name)
						assert.Equal(t, testExt, outfile.Extension)
						assert.Equal(t, pkg.FileStatusComplete, outfile.Status)
						assert.Equal(t, pkg.CreateUserFacingError("", nil), outfile.ErrorDetails)
						assert.Equal(t, pkg.OwnerNone, outfile.Owner)
						assert.Equal(t, pkg.OwnerNone, job.Owner)
					}
				}
			}
			repoMock.AssertExpectations(t)
			senderMock.AssertExpectations(t)
		})
	}
	// test file clean up
	_ = os.RemoveAll(testDir)
}
