/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import (
	"aicsd/pkg"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/clients/redis/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	Hostname = "gateway"
)

func TestRedisDB_New(t *testing.T) {
	mockRedisClient := mocks.DBClient{}
	mockConn := mocks.Conn{}
	mockRedisClient.On("TestConnection").Return(&mockConn, nil)
	mockConn.On("Close").Return(nil)
	_, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
	require.NoError(t, err)
}

func TestRedisDB_Create(t *testing.T) {

	validJob := helpers.CreateTestJob(pkg.OwnerDataOrg, Hostname)
	validJob.Id = ""

	jobWithId := validJob
	jobWithId.Id = uuid.NewString()

	jobStr, _ := json.Marshal(jobWithId)

	tests := []struct {
		Name             string
		Job              *types.Job
		ConnJobKey       interface{}
		ConnInputFileErr error
		ConnJob          interface{}
		ConnJobErr       error
		ConnExecErr      error
		ExpectedStatus   string
		ExpectedErr      error
	}{
		{"happy path: new", &validJob, nil, redigo.ErrNil, nil, nil, nil, StatusCreated, nil},
		{"happy path: exists", &validJob, []uint8(jobWithId.Id), nil, jobStr, nil, nil, StatusExists, nil},
		{"empty job", &types.Job{}, nil, nil, nil, nil, nil, StatusNone, pkg.ErrJobInvalid},
		{"input file query conn error", &validJob, nil, redigo.ErrPoolExhausted, nil, nil, nil, StatusNone, redigo.ErrPoolExhausted},
		{"job query conn error", &validJob, []uint8(jobWithId.Id), nil, nil, redigo.ErrPoolExhausted, nil, StatusNone, redigo.ErrPoolExhausted},
		{"new job exec error", &validJob, nil, redigo.ErrNil, nil, nil, redigo.ErrPoolExhausted, StatusNone, redigo.ErrPoolExhausted},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HGET, redis.KeyInputFile, mock.Anything).Return(test.ConnJobKey, test.ConnInputFileErr)
			mockConn.On("Do", redis.HGET, redis.KeyJob, mock.Anything).Return(test.ConnJob, test.ConnJobErr)
			mockConn.On("Send", redis.MULTI).Return(nil)
			mockConn.On("Send", redis.SET, mock.Anything, mock.Anything).Return(nil)
			mockConn.On("Send", redis.HSET, redis.KeyJob, mock.Anything, mock.Anything).Return(nil)
			mockConn.On("Send", redis.HSET, redis.KeyInputFile, mock.Anything, mock.Anything).Return(nil)
			mockConn.On("Send", redis.HSET, redis.CreateKey(redis.KeyOwner, pkg.OwnerDataOrg), mock.Anything, "").Return(nil)
			mockConn.On("Do", redis.EXEC).Return(nil, test.ConnExecErr)
			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualStatus, actualJob, actualErr := persistence.Create(*test.Job)
			require.Equal(t, test.ExpectedStatus, actualStatus)
			if test.ExpectedErr != nil {
				require.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
			}
			if test.ExpectedStatus == StatusCreated || test.ExpectedStatus == StatusExists {
				require.NotNil(t, actualJob.Id)
			}
		})
	}
}

func TestRedisDB_Update(t *testing.T) {
	var expectedReply, expectedReplyOwner, badReply []interface{}
	validJob := helpers.CreateTestJob(pkg.OwnerDataOrg, Hostname)
	validJob.Id = uuid.NewString()
	jobStr, err := json.Marshal(validJob)
	require.NoError(t, err)

	changeStatusFields := make(map[string]interface{})
	changeStatusFields[types.JobStatus] = pkg.StatusNoPipeline

	changeOwnerFields := make(map[string]interface{})
	changeOwnerFields[types.JobOwner] = pkg.OwnerNone

	changeNestedFields := make(map[string]interface{})
	changeNestedFields[types.JobOwner] = pkg.OwnerTaskLauncher
	changeNestedFields[types.JobInputFileHost] = "oem"

	bogusFields := make(map[string]interface{})
	bogusFields["bogus"] = "bogus"
	nestBogusFields := make(map[string]interface{})
	nestBogusFields["bogus.bogus"] = "bogus"
	bogusLastUpdatedFields := make(map[string]interface{})
	bogusLastUpdatedFields["LastUpdated"] = "bogus"

	expectedReply = append(expectedReply, "OK")

	expectedReplyOwner = append(expectedReplyOwner, int64(0))
	expectedReplyOwner = append(expectedReplyOwner, int64(1))
	expectedReplyOwner = append(expectedReplyOwner, int64(1))
	expectedReplyOwner = append(expectedReplyOwner, "OK")

	badReply = append(badReply, int64(0))
	badReply = append(badReply, int64(0))
	badReply = append(badReply, int64(1))
	badReply = append(badReply, "")

	tests := []struct {
		Name          string
		Id            string
		JobFields     map[string]interface{}
		MockExists    interface{}
		MockExistsErr error
		MockJob       interface{}
		MockGetErr    error
		MockWatchErr  error
		ChangeOwner   bool
		MockExecReply interface{}
		MockExecErr   error
		ExpectedErr   error
	}{
		{"happy path - change status", validJob.Id, changeStatusFields, []uint8("1"), nil, jobStr, nil, nil, false, expectedReply, nil, nil},
		{"happy path - change owner", validJob.Id, changeOwnerFields, []uint8("1"), nil, jobStr, nil, nil, true, expectedReplyOwner, nil, nil},
		{"happy path - change nested value", validJob.Id, changeNestedFields, []uint8("1"), nil, jobStr, nil, nil, true, expectedReplyOwner, nil, nil},
		{"empty id", "", nil, nil, nil, nil, nil, nil, false, nil, nil, pkg.ErrJobIdEmpty},
		{"empty job fields", "bogus", make(map[string]interface{}), nil, nil, nil, nil, nil, false, nil, nil, errors.New("no job fields provided to update")},
		{"job does not exist", "bogus", changeStatusFields, []uint8("0"), nil, nil, nil, nil, false, nil, nil, fmt.Errorf(pkg.ErrJobIdNotFound, "bogus")},
		{"job exist connection error", validJob.Id, changeStatusFields, []uint8("0"), redigo.ErrPoolExhausted, jobStr, nil, nil, false, expectedReply, nil, fmt.Errorf(pkg.ErrFmtRetrieving, validJob.Id)},
		{"job get not found", validJob.Id, changeStatusFields, []uint8("1"), nil, jobStr, redigo.ErrNil, nil, false, expectedReply, nil, fmt.Errorf(pkg.ErrJobIdNotFound, validJob.Id)},
		{"job get failed", validJob.Id, changeStatusFields, []uint8("1"), nil, jobStr, redigo.ErrPoolExhausted, nil, false, expectedReply, nil, pkg.ErrRetrieving},
		{"job watch error", validJob.Id, changeStatusFields, []uint8("1"), nil, jobStr, nil, redigo.ErrPoolExhausted, false, expectedReply, nil, fmt.Errorf(pkg.ErrFmtRedisWatchFailed, validJob.Id)},
		{"job unmarshal failed", validJob.Id, changeStatusFields, []uint8("1"), nil, []uint8("bogus"), nil, nil, false, expectedReply, nil, pkg.ErrUnmarshallingJob},
		{"job update helper bogus key", validJob.Id, bogusFields, []uint8("1"), nil, jobStr, nil, nil, false, expectedReply, nil, errors.New("entry bogus does not exist")},
		{"job update helper nested bogus key", validJob.Id, nestBogusFields, []uint8("1"), nil, jobStr, nil, nil, false, expectedReply, nil, errors.New("entry bogus.bogus does not exist")},
		{"change owner bad reply", validJob.Id, changeOwnerFields, []uint8("1"), nil, jobStr, nil, nil, true, badReply, nil, errors.New("unexpected value in reply")},
		{"change owner reply fail", validJob.Id, changeOwnerFields, []uint8("1"), nil, jobStr, nil, nil, true, expectedReply, redigo.ErrPoolExhausted, pkg.ErrUpdating},
		{"job fields is not marshal-able", validJob.Id, bogusLastUpdatedFields, []uint8("1"), nil, jobStr, nil, nil, true, nil, nil, pkg.ErrUnmarshallingJob},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)

			mockConn.On("Do", redis.HEXISTS, redis.KeyJob, mock.Anything).Return(test.MockExists, test.MockExistsErr)
			mockConn.On("Do", redis.HGET, redis.KeyJob, test.Id).Return(test.MockJob, test.MockGetErr)

			lockKey := redis.CreateKey(redis.KeyLock, redis.KeyJob, test.Id)
			mockConn.On("Do", redis.WATCH, lockKey).Return(nil, test.MockWatchErr)

			mockConn.On("Send", redis.MULTI).Return(nil)
			mockConn.On("Send", redis.HSET, redis.KeyJob, test.Id, mock.Anything).Return(nil)
			if test.ChangeOwner {
				mockConn.On("Send", redis.HDEL, mock.Anything, test.Id).Return(nil)
				mockConn.On("Send", redis.HSET, mock.Anything, test.Id, "").Return(nil)
			}
			mockConn.On("Send", redis.SET, lockKey, "").Return(nil)
			mockConn.On("Do", redis.EXEC).Return(test.MockExecReply, test.MockExecErr)

			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualJob, actualErr := persistence.Update(test.Id, test.JobFields)
			if test.ExpectedErr != nil {
				require.NotNil(t, actualErr)
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			assert.NotNil(t, actualJob)
			mockConn.AssertExpectations(t)
			mockRedisClient.AssertExpectations(t)
		})
	}
}

func TestRedisDB_Delete(t *testing.T) {
	var expectedSingleHash, expectedMultiHash, expectedBadHash, expectedReply, badReply []interface{}
	job := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, Hostname)

	expectedSingleHash = append(expectedSingleHash, []uint8(redis.CreateKey("oem", job.InputFile.DirName, job.InputFile.Name)))
	expectedSingleHash = append(expectedSingleHash, []uint8(job.Id))

	expectedMultiHash = append(expectedMultiHash, []uint8(redis.CreateKey("oem", job.InputFile.DirName, "input2.tiff")))
	expectedMultiHash = append(expectedMultiHash, []uint8("2"))
	expectedMultiHash = append(expectedMultiHash, []uint8(redis.CreateKey("oem", job.InputFile.DirName, job.InputFile.Name)))
	expectedMultiHash = append(expectedMultiHash, []uint8(job.Id))

	expectedBadHash = append(expectedBadHash, []uint8(redis.CreateKey("oem", job.InputFile.DirName, job.InputFile.Name)))
	expectedBadHash = append(expectedBadHash, []uint8(uuid.NewString()))

	expectedReply = append(expectedReply, []uint8("1"))
	expectedReply = append(expectedReply, []uint8("1"))
	expectedReply = append(expectedReply, []uint8("1"))
	expectedReply = append(expectedReply, []uint8("1"))

	badReply = append(badReply, []uint8("1"))
	badReply = append(badReply, []uint8("0"))
	badReply = append(badReply, []uint8("1"))
	badReply = append(badReply, []uint8("1"))

	tests := []struct {
		Name        string
		Id          string
		Job         types.Job
		GetJobErr   error
		GetAllHash  []interface{}
		GetAllErr   error
		WatchErr    error
		ExecReply   []interface{}
		ExecErr     error
		ExpectedErr error
	}{
		{"happy path single job in get all", job.Id, job, nil, expectedSingleHash, nil, nil, expectedReply, nil, nil},
		{"happy path multiple jobs in get all", job.Id, job, nil, expectedMultiHash, nil, nil, expectedReply, nil, nil},
		{"empty job id", "", job, nil, nil, nil, nil, nil, nil, pkg.ErrJobIdEmpty},
		{"job get connection error", job.Id, job, redigo.ErrPoolExhausted, nil, nil, nil, nil, nil, pkg.ErrRetrieving},
		{"job get all id not found", job.Id, job, nil, expectedBadHash, nil, nil, nil, nil, fmt.Errorf(pkg.ErrJobIdNotFound, job.Id)},
		{"job get all connection error", job.Id, job, nil, nil, redigo.ErrPoolExhausted, nil, nil, nil, pkg.ErrRetrieving},
		{"job watch connection error", job.Id, job, nil, expectedSingleHash, nil, redigo.ErrPoolExhausted, nil, nil, fmt.Errorf(pkg.ErrFmtRedisWatchFailed, job.Id)},
		{"exec connection error", job.Id, job, nil, expectedSingleHash, nil, nil, nil, redigo.ErrPoolExhausted, fmt.Errorf(pkg.ErrFmtJobDelete, job.Id)},
		{"exec reply error", job.Id, job, nil, expectedSingleHash, nil, nil, badReply, nil, fmt.Errorf(pkg.ErrFmtJobDelete, job.Id)},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			jobStr, err := json.Marshal(job)
			require.NoError(t, err)
			ownerKey := redis.CreateKey(redis.KeyOwner, test.Job.Owner)
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HGET, redis.KeyJob, test.Id).Return(jobStr, test.GetJobErr)
			mockConn.On("Do", redis.HGETALL, redis.KeyInputFile).Return(test.GetAllHash, test.GetAllErr)
			mockConn.On("Do", redis.WATCH, mock.Anything).Return(nil, test.WatchErr)
			mockConn.On("Send", redis.MULTI).Return(nil)
			mockConn.On("Send", redis.HDEL, redis.KeyJob, test.Id).Return(nil)
			mockConn.On("Send", redis.HDEL, ownerKey, test.Id).Return(nil)
			mockConn.On("Send", redis.HDEL, redis.KeyInputFile, mock.Anything).Return(nil)
			mockConn.On("Send", redis.DEL, mock.Anything).Return(nil)
			mockConn.On("Do", redis.EXEC).Return(test.ExecReply, test.ExecErr)

			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualErr := persistence.Delete(test.Id)
			if test.ExpectedErr != nil {
				require.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}
}

func TestRedisDB_GetAll(t *testing.T) {
	job := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, Hostname)}
	jobs := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, Hostname), helpers.CreateTestJob(pkg.OwnerTaskLauncher, Hostname)}
	jobs[1].Id = "2"
	jobs[1].InputFile.Name = "new-image.tiff"
	jobs[1].InputFile.Attributes = map[string]string{
		"User":        "Scientist3",
		"BatchNumber": "13",
	}

	tests := []struct {
		Name        string
		Jobs        []types.Job
		ConnJobErr  error
		ExpectedErr error
	}{
		{"happy path - no jobs", []types.Job{}, nil, nil},
		{"happy path - single job", job, nil, nil},
		{"happy path - multiple jobs", jobs, nil, nil},
		{"redis connection failed", []types.Job{}, redigo.ErrPoolExhausted, redigo.ErrPoolExhausted},
		{"redis empty jobs db", []types.Job{}, redigo.ErrNil, pkg.ErrJobsEmpty},
		{"unmarshal job failed", nil, nil, pkg.ErrUnmarshallingJob},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var expectedJobs []interface{}
			var err error
			// build up the interface as it is expected from redis conn.Do
			if test.Jobs != nil {
				expectedJobs, err = createInterfaceFromJobs(test.Jobs)
			} else {
				// if test.Jobs is nil, add bogus that can't be unmarshalled
				expectedJobs = append(expectedJobs, []uint8("bogus"))
			}

			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HVALS, redis.KeyJob).Return(expectedJobs, test.ConnJobErr)

			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualJobs, actualErr := persistence.GetAll()
			if test.ExpectedErr != nil {
				require.Error(t, actualErr)
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			assert.Equal(t, test.Jobs, actualJobs)
			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}
}

func TestRedisDB_GetById(t *testing.T) {
	job := helpers.CreateTestJob(pkg.OwnerTaskLauncher, Hostname)
	jobStr, err := json.Marshal(job)
	require.NoError(t, err)
	tests := []struct {
		Name        string
		Id          string
		Job         interface{}
		ConnJobErr  error
		ExpectedErr error
	}{
		{"happy path", job.Id, jobStr, nil, nil},
		{"empty job id", "", "", nil, pkg.ErrJobIdEmpty},
		{"bogus job id", "bogus", "", redigo.ErrNil, fmt.Errorf(pkg.ErrJobIdNotFound, "bogus")},
		{"connection error", "bogus", "", redigo.ErrPoolExhausted, pkg.ErrRetrieving},
		{"unmarshall job fail", "bogus", "bogus", nil, pkg.ErrUnmarshallingJob},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HGET, redis.KeyJob, test.Id).Return(test.Job, test.ConnJobErr)

			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualJob, actualErr := persistence.GetById(test.Id)
			if test.ExpectedErr != nil {
				require.Error(t, actualErr)
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			// check that id matches
			assert.Equal(t, test.Id, actualJob.Id)
			// check that job matches
			actualJobStr, err := json.Marshal(job)
			require.NoError(t, err)
			assert.Equal(t, test.Job, actualJobStr)
			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}
}

func TestRedisDB_GetByOwner(t *testing.T) {
	job := []types.Job{helpers.CreateTestJob(pkg.OwnerFileRecvGateway, Hostname)}
	jobs := []types.Job{helpers.CreateTestJob(pkg.OwnerFileSenderOem, Hostname),
		helpers.CreateTestJob(pkg.OwnerFileSenderOem, Hostname)}
	jobs[1].Id = "2"
	jobs[1].InputFile.Name = "new-image.tiff"

	tests := []struct {
		Name          string
		Owner         string
		Jobs          []types.Job
		ConnGetErr    error
		ConnGetAllErr error
		ExpectedErr   error
	}{
		{"happy path - no jobs", pkg.OwnerNone, []types.Job{}, nil, nil, nil},
		{"happy path - single job", job[0].Owner, job, nil, nil, nil},
		{"happy path - multiple jobs", jobs[0].Owner, jobs, nil, nil, nil},
		{"bogus owner", "bogus", nil, nil, nil, fmt.Errorf(pkg.ErrFmtInvalidInput, "bogus", "proper owner name")},
		{"redis connection failed for get all", pkg.OwnerNone, []types.Job{}, nil, redigo.ErrPoolExhausted, pkg.ErrCallingGet},
		{"jobs db is empty", pkg.OwnerNone, []types.Job{}, nil, redigo.ErrNil, pkg.ErrJobsEmpty},
		{"redis connection failed for get single job", job[0].Owner, job, redigo.ErrPoolExhausted, nil, pkg.ErrRetrieving},
		{"redis connection failed for get multiple jobs", jobs[0].Owner, jobs, redigo.ErrPoolExhausted, nil, pkg.ErrRetrieving},
		{"redis connection no results for get", job[0].Owner, job, redigo.ErrNil, nil, redigo.ErrNil},
		{"unmarshal job failed", pkg.OwnerNone, nil, nil, nil, pkg.ErrUnmarshallingJob},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var expectedJobIds []interface{}
			var expectedJob interface{}
			var err error
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			// build and set mocks for HGET
			if test.Jobs != nil {
				for _, data := range test.Jobs {
					expectedJobIds = append(expectedJobIds, []uint8(data.Id))
					expectedJob, err = json.Marshal(data)
					mockConn.On("Do", redis.HGET, redis.KeyJob, data.Id).Return(expectedJob, test.ConnGetErr)
				}
			} else {
				data := "bogus"
				expectedJobIds = append(expectedJobIds, []uint8(data))
				expectedJob = []uint8(data)
				mockConn.On("Do", redis.HGET, redis.KeyJob, data).Return(expectedJob, test.ConnGetErr)
			}
			ownerKey := redis.CreateKey(redis.KeyOwner, test.Owner)
			mockConn.On("Do", redis.HKEYS, ownerKey).Return(expectedJobIds, test.ConnGetAllErr)

			// create redisdb persistence instance
			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualJobs, actualErr := persistence.GetByOwner(test.Owner)
			if test.ExpectedErr != nil {
				require.Error(t, actualErr)
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			assert.Equal(t, test.Jobs, actualJobs)
			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}
}

func TestRedisDB_updateHelper(t *testing.T) {
	var value interface{}
	var jobMap map[string]interface{}
	var actualJob types.Job
	job := helpers.CreateTestJob(pkg.OwnerFileSenderGateway, Hostname)
	jobStr, err := json.Marshal(job)
	require.NoError(t, err)

	err = json.Unmarshal(jobStr, &jobMap)
	require.NoError(t, err)
	key := "InputFile.Hostname"
	keys := strings.Split(key, ".")
	value = "oem"

	err = updateHelper(&jobMap, keys, value)
	require.NoError(t, err)

	actualJobStr, err := json.Marshal(jobMap)
	require.NoError(t, err)
	assert.NotEqual(t, jobStr, actualJobStr)
	err = json.Unmarshal(actualJobStr, &actualJob)
	require.NoError(t, err)
	assert.Equal(t, actualJob.InputFile.Hostname, value)
}

func createInterfaceFromJobs(jobs []types.Job) ([]interface{}, error) {
	var result []interface{}
	for _, job := range jobs {
		data, err := json.Marshal(job)
		if err != nil {
			return nil, err
		}
		result = append(result, data)
	}
	return result, nil
}
