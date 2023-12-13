/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import (
	"encoding/json"
	"fmt"
	"testing"

	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/clients/redis/mocks"
	"aicsd/pkg/types"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRedisDB_New makes a connection to RedisDB if the container is up otherwise it fails
func TestRedisDB_New(t *testing.T) {
	mockRedisClient := mocks.DBClient{}
	mockConn := mocks.Conn{}
	mockRedisClient.On("TestConnection").Return(&mockConn, nil)
	mockConn.On("Close").Return(nil)
	_, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
	require.NoError(t, err)
}

// TestRedisDB_Create stores a new task to RedisDB else it fails
func TestRedisDB_Create(t *testing.T) {
	valid := taskPkg.CreateTestTask("", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")

	invalidWithEmptyDescription := taskPkg.CreateTestTask("", "", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")
	invalidWithEmptyJobSelector := taskPkg.CreateTestTask("", "Count Cells", "", "100")
	invalidWithEmptyPipelineId := taskPkg.CreateTestTask("", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "")

	tests := []struct {
		Name          string
		Task          *types.Task
		ConnDoErr     error
		ExpectedError error
	}{
		{"happy path: new", &valid, nil, nil},
		{"empty description", &invalidWithEmptyDescription, nil, taskPkg.ErrTaskEmptyDescription},
		{"empty job selector", &invalidWithEmptyJobSelector, nil, taskPkg.ErrTaskEmptyJobSelector},
		{"empty pipeline id", &invalidWithEmptyPipelineId, nil, taskPkg.ErrTaskEmptyPipelineId},
		{"Do connection err", &valid, redigo.ErrPoolExhausted, redigo.ErrPoolExhausted},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HSET, mock.Anything, mock.Anything, mock.Anything).Return(nil, test.ConnDoErr)

			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)

			actualTaskId, actualErr := persistence.Create(*test.Task)
			if test.ExpectedError != nil {
				assert.Contains(t, actualErr.Error(), test.ExpectedError.Error())

				// Call mockConn.asserts for Do, only when error is generated from it
				if test.ConnDoErr != nil {
					mockRedisClient.AssertCalled(t, "GetConnection", mock.Anything)
					mockConn.AssertCalled(t, "Do", redis.HSET, mock.Anything, mock.Anything, mock.Anything)
				}

			} else {
				require.NoError(t, actualErr)
				assert.NotEmpty(t, actualTaskId)
				mockRedisClient.AssertCalled(t, "GetConnection", mock.Anything)
				mockConn.AssertCalled(t, "Do", redis.HSET, mock.Anything, mock.Anything, mock.Anything)
			}

			mockRedisClient.AssertCalled(t, "TestConnection", mock.Anything)
			mockConn.AssertCalled(t, "Close", mock.Anything)

		})
	}
}

// TestRedisDB_GetById gets a task from RedisDB based on task Id else it fails
func TestRedisDB_GetById(t *testing.T) {
	valid := taskPkg.CreateTestTask("1", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")

	jsonValid, err := json.Marshal(valid)
	require.NoError(t, err)

	tests := []struct {
		Name        string
		HGetTaskArg []byte
		TaskId      string
		HGetErr     error
		ExpectedErr error
	}{
		{"happy path: get by id", []byte(jsonValid), "1", nil, nil},
		{"task id not found", nil, "1", redigo.ErrNil, redigo.ErrNil},
		{"task id is empty", []byte(jsonValid), "", nil, taskPkg.ErrTaskIdEmpty},
		{"Do HGET connection err", []byte(jsonValid), "1", redigo.ErrPoolExhausted, redigo.ErrPoolExhausted},
		{"unmarshal err", nil, "1", nil, pkg.ErrJSONMarshalErr},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HGET, mock.Anything, valid.Id).Return(test.HGetTaskArg, test.HGetErr)

			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)

			actualTask, actualErr := persistence.GetById(test.TaskId)

			if test.ExpectedErr != nil {
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())

				// Call mockConn.asserts for Do, only when error is generated from it
				if test.HGetErr != nil {
					mockRedisClient.AssertCalled(t, "GetConnection", mock.Anything)
					mockConn.AssertCalled(t, "Do", redis.HGET, mock.Anything, test.TaskId)
				}

			} else {
				require.NoError(t, actualErr)
				assert.NotEmpty(t, actualTask)
				mockRedisClient.AssertCalled(t, "GetConnection", mock.Anything)
				mockConn.AssertCalled(t, "Do", redis.HGET, mock.Anything, test.TaskId)
			}

			mockRedisClient.AssertCalled(t, "TestConnection", mock.Anything)
			mockConn.AssertCalled(t, "Close", mock.Anything)

		})
	}

}

// TestRedisDB_GetAll gets all tasks from RedisDB else it fails
func TestRedisDB_GetAll(t *testing.T) {

	singleTask := []types.Task{taskPkg.CreateTestTask("1", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")}
	task2 := taskPkg.CreateTestTask("2", "Count Cells", `{ "==" : [ { "var" : "Id" }, "2" ] }`, "200")
	task2.ModelParameters = map[string]string{"Gamma": "255"}
	multipleTasks := append(singleTask, task2)

	tests := []struct {
		Name          string
		ExpectedTasks []types.Task
		HValsErr      error
		ExpectedErr   error
	}{
		{"happy path: single task", singleTask, nil, nil},
		{"happy path: multiple tasks", multipleTasks, nil, nil},
		{"no task returned", nil, redigo.ErrNil, taskPkg.ErrTaskEmpty},
		{"Do HVALS connection err", multipleTasks, redigo.ErrPoolExhausted, redigo.ErrPoolExhausted},
		{"unmarshal err", nil, nil, pkg.ErrJSONMarshalErr},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var tasks []interface{}
			var err error
			if test.ExpectedTasks != nil {
				tasks, err = taskPkg.CreateJsonTasks(test.ExpectedTasks)
				require.NoError(t, err)
			} else {
				tasks = append(tasks, nil)
			}

			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HVALS, redis.KeyTask).Return(tasks, test.HValsErr)

			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)

			actualTasks, actualErr := persistence.GetAll()

			if test.ExpectedErr != nil {
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())

			} else {
				require.NoError(t, actualErr)
				assert.NotEmpty(t, actualTasks)
				assert.Equal(t, test.ExpectedTasks, actualTasks)
			}

			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)

		})
	}
}

// TestRedisDB_Delete removes task from the RedisDB based on specified task id else it fails
func TestRedisDB_Delete(t *testing.T) {

	tests := []struct {
		Name          string
		TaskId        string
		NumDeletes    int
		ExpectedError error
	}{
		{"happy path", "1", 1, nil},
		{"empty id", "", 0, taskPkg.ErrTaskIdEmpty},
		{"delete fail", "1", 0, taskPkg.ErrDeleteTask},
		{"no task deleted", "1", 0, taskPkg.ErrNoTaskDeleted},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HDEL, redis.KeyTask, test.TaskId).Return(test.NumDeletes, test.ExpectedError)

			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualErr := persistence.Delete(test.TaskId)
			if test.ExpectedError != nil {
				require.Contains(t, actualErr.Error(), test.ExpectedError.Error())
				return
			}

			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}

}

func TestRedisDB_Update(t *testing.T) {
	var expectedReply, badReply []interface{}

	validTask := taskPkg.CreateTestTask("1", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")
	taskNoId := taskPkg.CreateTestTask("", "Count Cells", `{ "==" : [ { "var" : "Id" }, "1" ] }`, "100")

	jsonValid, err := json.Marshal(validTask)
	require.NoError(t, err)

	expectedReply = append(expectedReply, int64(0))
	expectedReply = append(expectedReply, "OK")

	badReply = append(badReply, int64(0))
	badReply = append(badReply, "")

	tests := []struct {
		Name          string
		Task          types.Task
		MockExists    interface{}
		MockExistsErr error
		HGetTaskArg   interface{}
		MockGetErr    error
		MockWatchErr  error
		MockExecReply []interface{}
		ExpectedErr   error
	}{
		{"happy path", validTask, []uint8("1"), nil, jsonValid, nil, nil, expectedReply, nil},
		{"empty id", taskNoId, nil, nil, nil, nil, nil, expectedReply, taskPkg.ErrTaskIdEmpty},
		{"task doesn't exist", validTask, []uint8("0"), nil, jsonValid, nil, nil, expectedReply, fmt.Errorf(taskPkg.ErrFmtTaskIdNotFound, validTask.Id)},
		{"task exists error", validTask, []uint8("0"), redigo.ErrPoolExhausted, jsonValid, nil, nil, expectedReply, taskPkg.ErrTaskRetrieval},
		{"task get error", validTask, []uint8("1"), nil, jsonValid, redigo.ErrPoolExhausted, nil, expectedReply, taskPkg.ErrTaskRetrieval},
		{"task get not found", validTask, []uint8("1"), nil, jsonValid, redigo.ErrNil, nil, expectedReply, fmt.Errorf(taskPkg.ErrFmtTaskIdNotFound, validTask.Id)},
		{"task update error", validTask, []uint8("1"), nil, jsonValid, nil, nil, expectedReply, taskPkg.ErrTaskCreation},
		{"lock db error", validTask, []uint8("1"), nil, jsonValid, nil, redigo.ErrPoolExhausted, expectedReply, fmt.Errorf(pkg.ErrFmtRedisWatchFailed, validTask.Id)},
		{"bad reply error", validTask, []uint8("1"), nil, jsonValid, nil, nil, badReply, fmt.Errorf("unexpected value in reply")},
		{"unmarshal error", validTask, []uint8("1"), nil, []uint8("bogus"), nil, nil, expectedReply, taskPkg.ErrUnmarshallingTask},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			// set mocks
			mockRedisClient := mocks.DBClient{}
			mockConn := mocks.Conn{}
			mockRedisClient.On("TestConnection").Return(&mockConn, nil)
			mockRedisClient.On("GetConnection").Return(&mockConn)
			mockConn.On("Close").Return(nil)
			mockConn.On("Do", redis.HEXISTS, redis.KeyTask, test.Task.Id).Return(test.MockExists, test.MockExistsErr)
			lockKey := redis.CreateKey(redis.KeyLock, redis.KeyTask, test.Task.Id)
			mockConn.On("Do", redis.WATCH, lockKey).Return(nil, test.MockWatchErr)
			mockConn.On("Do", redis.HGET, redis.KeyTask, test.Task.Id).Return(test.HGetTaskArg, test.MockGetErr)
			mockConn.On("Send", redis.MULTI).Return(nil)
			mockConn.On("Send", redis.HSET, redis.KeyTask, test.Task.Id, mock.Anything).Return(nil)
			mockConn.On("Send", redis.SET, lockKey, "").Return(nil)
			mockConn.On("Do", redis.EXEC).Return(test.MockExecReply, test.ExpectedErr)

			persistence, err := NewRedisDB(logger.MockLogger{}, &mockRedisClient)
			require.NoError(t, err)
			actualErr := persistence.Update(test.Task)
			if test.ExpectedErr != nil {
				require.Error(t, actualErr)
				assert.Contains(t, actualErr.Error(), test.ExpectedErr.Error())
				return
			}
			require.NoError(t, actualErr)
			mockRedisClient.AssertExpectations(t)
			mockConn.AssertExpectations(t)
		})
	}
}
