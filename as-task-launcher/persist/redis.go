/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import (
	"encoding/json"
	"fmt"

	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

type RedisDB struct {
	lc          logger.LoggingClient
	redisClient redis.DBClient
}

// NewRedisDB returns a persistence interface that uses redis database
// it will test the connection and return an error if the connection fails
func NewRedisDB(lc logger.LoggingClient, redisClient redis.DBClient) (Persistence, error) {
	persistence := RedisDB{
		lc:          lc,
		redisClient: redisClient,
	}
	conn, err := persistence.redisClient.TestConnection()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	return persistence, nil
}

// Create stores a new task object in the Redis DB
// Task is stored in the {key,field,value} format where key is "task", field is task.Id & value is {taskObject}
func (rdb RedisDB) Create(task types.Task) (string, error) {

	if task.JobSelector == "" {
		return "", taskPkg.ErrTaskEmptyJobSelector
	}

	if task.PipelineId == "" {
		return "", taskPkg.ErrTaskEmptyPipelineId
	}

	if task.Description == "" {
		return "", taskPkg.ErrTaskEmptyDescription
	}

	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	task.Id = uuid.NewString()
	task.SetLastUpdated()

	jsonTask, err := json.Marshal(task)
	if err != nil {
		return "", werrors.WrapErr(err, taskPkg.ErrMarshallingTask)
	}

	_, err = conn.Do(redis.HSET, redis.KeyTask, task.Id, jsonTask)
	if err != nil {
		return "", werrors.WrapErr(err, taskPkg.ErrTaskCreation)
	}

	return task.Id, nil

}

func (rdb RedisDB) Update(task types.Task) error {
	// check id exists - not empty
	if task.Id == "" {
		return taskPkg.ErrTaskIdEmpty
	}

	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	// see if the task exists
	exists, err := redigo.Bool(conn.Do(redis.HEXISTS, redis.KeyTask, task.Id))
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
	}
	if !exists {
		return fmt.Errorf(taskPkg.ErrFmtTaskIdNotFound, task.Id)
	}

	lockKey := redis.CreateKey(redis.KeyLock, redis.KeyTask, task.Id)
	_, err = conn.Do(redis.WATCH, lockKey)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRedisWatchFailed, task.Id)
	}

	data, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyTask, task.Id))
	if err == redigo.ErrNil {
		return werrors.WrapMsgf(err, taskPkg.ErrFmtTaskIdNotFound, task.Id)
	}
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
	}

	var existingTask types.Task
	err = json.Unmarshal(data, &existingTask)
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrUnmarshallingTask)
	}

	existingTask.ReplaceTask(task)

	jsonTask, err := json.Marshal(existingTask)
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrMarshallingTask)
	}

	conn.Send(redis.MULTI)
	conn.Send(redis.HSET, redis.KeyTask, task.Id, jsonTask)
	conn.Send(redis.SET, lockKey, "")
	reply, err := redigo.Values(conn.Do(redis.EXEC))
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrTaskCreation)
	}
	if reply[len(reply)-1] != "OK" {
		return fmt.Errorf("unexpected value in reply got %s instead of OK", reply[len(reply)-1])
	}
	return nil
}

func (rdb RedisDB) Delete(id string) error {

	if id == "" {
		return taskPkg.ErrTaskIdEmpty
	}

	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	numDeletes, err := redigo.Int(conn.Do(redis.HDEL, redis.KeyTask, id))
	if err != nil {
		return werrors.WrapErr(err, taskPkg.ErrDeleteTask)
	}
	if numDeletes == 0 {
		return taskPkg.ErrNoTaskDeleted
	}

	return nil
}

// GetById returns task object corresponding to a particular task Id
func (rdb RedisDB) GetById(id string) (types.Task, error) {

	if id == "" {
		return types.Task{}, taskPkg.ErrTaskIdEmpty
	}

	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	taskBytes, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyTask, id))

	if err == redigo.ErrNil {
		return types.Task{}, werrors.WrapMsgf(err, taskPkg.ErrFmtTaskIdNotFound, id)
	}

	if err != nil {
		return types.Task{}, werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
	}

	task := types.Task{}
	err = json.Unmarshal(taskBytes, &task)

	if err != nil {
		return types.Task{}, werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
	}

	return task, nil
}

// GetAll returns all tasks from the Redis DB
func (rdb RedisDB) GetAll() ([]types.Task, error) {
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	taskBytes, err := redigo.ByteSlices(conn.Do(redis.HVALS, redis.KeyTask))
	if err == redigo.ErrNil {
		return []types.Task{}, werrors.WrapErr(err, taskPkg.ErrTaskEmpty)

	}
	if err != nil {
		return []types.Task{}, werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
	}

	tasks := make([]types.Task, len(taskBytes))

	for i, taskJson := range taskBytes {
		task := types.Task{}
		err = json.Unmarshal(taskJson, &task)

		if err != nil {
			return []types.Task{}, werrors.WrapErr(err, taskPkg.ErrTaskRetrieval)
		}
		tasks[i] = task

	}

	return tasks, nil
}

func (rdb RedisDB) Disconnect() error {
	return rdb.redisClient.Disconnect()
}

func (rdb RedisDB) Filter(task types.Task) ([]types.Task, error) {
	return []types.Task{}, nil
}
