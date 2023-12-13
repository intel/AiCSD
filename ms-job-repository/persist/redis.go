/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aicsd/pkg"
	"aicsd/pkg/clients/redis"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

const (
	JobValidator = `{ "and" : [
						{ "!=" : [ {"var" : "InputFile.Hostname" }, "" ]},
						{ "!=" : [ {"var" : "InputFile.DirName" }, "" ]},
						{ "!=" : [ {"var" : "InputFile.Name" }, "" ]},
						{ "!=" : [ {"var" : "InputFile.Extension" }, "" ]}
					] }`
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

// Create returns a status (StatusExists or StatusCreated), the job, and an error (if one occurred)
// It creates a hash entry in redis db under the key job, the field {job.Id}, and the value str(job_object),
// a hash entry with the key input_file, the field hostname:dirname:filename, and value {job.Id},
// a hash entry with the key job|owner:{job.Owner}, the field {job.Id}, and an empty string
// and a set entry with the key lock:job:{job.Id} and value empty string.
// Note that the hash entry for the input file will always reference the originating system and will NOT be
// updated as the file moves across systems. This is for quick referencing for the creation of jobs
func (rdb RedisDB) Create(job types.Job) (string, types.Job, error) {
	// validate parameters
	if job.Id != "" {
		return StatusNone, types.Job{}, errors.New("id already specified")
	}

	isValid, err := helpers.ApplyJsonLogicToJob(job, JobValidator)
	if !isValid {
		return StatusNone, types.Job{}, pkg.ErrJobInvalid
	}

	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	// determine if job already exists
	// The key here uses the initial file location because that is where jobs are created
	inputFileInfo := redis.CreateKey(job.InputFile.Hostname, job.InputFile.DirName, job.InputFile.Name)
	checkJobKey, err := redigo.String(conn.Do(redis.HGET, redis.KeyInputFile, inputFileInfo))
	if err != nil && err != redigo.ErrNil {
		return StatusNone, types.Job{}, werrors.WrapErr(err, pkg.ErrRetrieving)
	}
	if err != redigo.ErrNil {
		checkJobValue, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyJob, checkJobKey))
		if err != nil && err != redigo.ErrNil {
			return StatusNone, types.Job{}, werrors.WrapErr(err, pkg.ErrRetrieving)
		}
		checkJob := types.Job{}
		err = json.Unmarshal(checkJobValue, &checkJob)
		if err != nil {
			return StatusNone, types.Job{}, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
		}

		return StatusExists, checkJob, nil
	}

	// job doesn't exist so set the job id and set in redis db
	job.Id = uuid.NewString()
	job.SetLastUpdated()
	job.ErrorDetails = pkg.CreateUserFacingError("", nil)
	jsonJob, err := json.Marshal(job)
	if err != nil {
		return StatusNone, types.Job{}, werrors.WrapErr(err, pkg.ErrMarshallingJob)
	}
	ownerKey := redis.CreateKey(redis.KeyOwner, job.Owner)
	lockKey := redis.CreateKey(redis.KeyLock, redis.KeyJob, job.Id)
	// job:{job_id}
	// key, field, value
	// job, {job_id}, {json string}
	_ = conn.Send(redis.MULTI)
	_ = conn.Send(redis.SET, lockKey, "")
	_ = conn.Send(redis.HSET, redis.KeyJob, job.Id, jsonJob)
	// input_file, host:dir:inputfile, job:{job_id}
	_ = conn.Send(redis.HSET, redis.KeyInputFile, inputFileInfo, job.Id)
	// owner:{owner_name}, job_id, empty string
	_ = conn.Send(redis.HSET, ownerKey, job.Id, "")
	_, err = conn.Do(redis.EXEC)
	if err != nil {
		return StatusNone, types.Job{}, werrors.WrapErr(err, pkg.ErrJobCreation)
	}
	return StatusCreated, job, nil
}

// Update retrieves the job id and modifies the entry using the given values
func (rdb RedisDB) Update(id string, jobFields map[string]interface{}) (types.Job, error) {
	// check id exists - not empty
	if id == "" {
		return types.Job{}, pkg.ErrJobIdEmpty
	}
	if len(jobFields) == 0 {
		return types.Job{}, errors.New("no job fields provided to update")
	}
	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	// see if the job exists
	exists, err := redigo.Bool(conn.Do(redis.HEXISTS, redis.KeyJob, id))
	if err != nil {
		return types.Job{}, werrors.WrapMsgf(err, pkg.ErrFmtRetrieving, id)
	}
	if !exists {
		return types.Job{}, fmt.Errorf(pkg.ErrJobIdNotFound, id)
	}
	// use redis WATCH to lock the database
	lockKey := redis.CreateKey(redis.KeyLock, redis.KeyJob, id)
	// TODO: add retry loop around WATCH if too many concurrent requests occur
	// WATCH commands always return ok, so ignore value
	_, err = conn.Do(redis.WATCH, lockKey)
	if err != nil {
		return types.Job{}, werrors.WrapMsgf(err, pkg.ErrFmtRedisWatchFailed, id)
	}
	// get the job to update
	data, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyJob, id))
	if err == redigo.ErrNil { // shouldn't hit this case
		return types.Job{}, werrors.WrapMsgf(err, pkg.ErrJobIdNotFound, id)
	}
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrRetrieving)
	}

	// apply map to update the job
	var jobMap map[string]interface{}
	err = json.Unmarshal(data, &jobMap)
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
	}
	// get the old owner to remove it if owner changed
	oldOwner := jobMap[types.JobOwner].(string)

	for key, value := range jobFields {
		keys := strings.Split(key, ".")
		err = updateHelper(&jobMap, keys, value)
		if err != nil {
			return types.Job{}, err
		}
	}

	// marshal the new job back to a []byte
	var updateJob types.Job
	data, err = json.Marshal(jobMap)
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrMarshallingJob)
	}

	// unmarshal the job to verify that jobFields were set properly
	err = json.Unmarshal(data, &updateJob)
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
	}

	// set the last updated and marshal the job back
	updateJob.SetLastUpdated()
	data, err = json.Marshal(updateJob)
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrMarshallingJob)
	}

	// set the new entry in redis and update the owner if necessary
	conn.Send(redis.MULTI)
	conn.Send(redis.HSET, redis.KeyJob, id, data)
	// check to see if the owner needs to be updated
	if oldOwner != updateJob.Owner {
		oldOwnerKey := redis.CreateKey(redis.KeyOwner, oldOwner)
		updateOwnerKey := redis.CreateKey(redis.KeyOwner, updateJob.Owner)
		conn.Send(redis.HDEL, oldOwnerKey, id)
		conn.Send(redis.HSET, updateOwnerKey, id, "")
	}
	conn.Send(redis.SET, lockKey, "")
	reply, err := redigo.Values(conn.Do(redis.EXEC))
	if err != nil {
		return types.Job{}, werrors.WrapErr(err, pkg.ErrUpdating)
	}
	// the last entry returned in reply is "OK"
	if reply[len(reply)-1] != "OK" {
		return types.Job{}, fmt.Errorf("unexpected value in reply got %s instead of OK", reply[len(reply)-1])
	}
	return updateJob, nil
}

// Delete deletes all entries associated with the given id
func (rdb RedisDB) Delete(id string) error {
	if id == "" {
		return pkg.ErrJobIdEmpty
	}
	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	// get the job to determine owner to delete
	job, err := rdb.GetById(id)
	if err != nil {
		return err
	}
	ownerKey := redis.CreateKey(redis.KeyOwner, job.Owner)
	// get the input file information to delete it
	inputFileField, err := rdb.getFromHashByJobId(conn, redis.KeyInputFile, id)
	if err != nil {
		return err
	}
	// use redis WATCH to lock the database
	lockKey := redis.CreateKey(redis.KeyLock, redis.KeyJob, id)
	// TODO: add retry loop around WATCH if too many concurrent requests occur
	// WATCH always returns ok, so ignore it.
	_, err = conn.Do(redis.WATCH, lockKey)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRedisWatchFailed, id)
	}
	_ = conn.Send(redis.MULTI)
	_ = conn.Send(redis.HDEL, redis.KeyJob, id)
	_ = conn.Send(redis.HDEL, ownerKey, id)
	_ = conn.Send(redis.HDEL, redis.KeyInputFile, inputFileField)
	_ = conn.Send(redis.DEL, lockKey)
	reply, err := redigo.Ints(conn.Do(redis.EXEC))
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtJobDelete, id)
	}
	err = verifyReply(reply, 1)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtJobDelete, id)
	}
	return nil
}

// GetAll retrieves all the jobs from redis
func (rdb RedisDB) GetAll() ([]types.Job, error) {
	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	result, err := redigo.ByteSlices(conn.Do(redis.HVALS, redis.KeyJob))
	if err == redigo.ErrNil {
		return []types.Job{}, werrors.WrapErr(err, pkg.ErrJobsEmpty)
	}
	if err != nil {
		return []types.Job{}, werrors.WrapMsg(err, "failed jobs GetAll from redis")
	}
	jobs := make([]types.Job, len(result))

	for i, jobJson := range result {
		job := types.Job{}
		err = json.Unmarshal(jobJson, &job)
		if err != nil {
			return jobs, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
		}
		jobs[i] = job
	}
	return jobs, nil
}

// GetById retrieves the job from redis matching the id passed in
func (rdb RedisDB) GetById(id string) (types.Job, error) {
	job := types.Job{}
	if id == "" {
		return job, pkg.ErrJobIdEmpty
	}
	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	data, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyJob, id))
	if err == redigo.ErrNil {
		return job, werrors.WrapMsgf(err, pkg.ErrJobIdNotFound, id)
	}
	if err != nil {
		return job, werrors.WrapErr(err, pkg.ErrRetrieving)
	}
	err = json.Unmarshal(data, &job)
	if err != nil {
		return job, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
	}

	return job, nil
}

// GetByOwner retrieves the jobs from redis matching the owner passed in
func (rdb RedisDB) GetByOwner(owner string) ([]types.Job, error) {
	if owner != pkg.OwnerNone && owner != pkg.OwnerFileWatcher && owner != pkg.OwnerDataOrg &&
		owner != pkg.OwnerFileSenderOem && owner != pkg.OwnerFileRecvGateway && owner != pkg.OwnerTaskLauncher &&
		owner != pkg.OwnerFileSenderGateway && owner != pkg.OwnerFileRecvOem {
		return nil, fmt.Errorf(pkg.ErrFmtInvalidInput, owner, "proper owner name")
	}

	// get a connection to redis db
	conn := rdb.redisClient.GetConnection()
	defer func() { _ = conn.Close() }()

	ownerKey := redis.CreateKey(redis.KeyOwner, owner)
	result, err := redigo.ByteSlices(conn.Do(redis.HKEYS, ownerKey))
	if err == redigo.ErrNil {
		return []types.Job{}, werrors.WrapErr(err, pkg.ErrJobsEmpty)
	}
	if err != nil {
		return []types.Job{}, werrors.WrapErr(err, pkg.ErrCallingGet)
	}
	jobs := make([]types.Job, len(result))

	for i, jobId := range result {
		job := types.Job{}
		jobJson, err := redigo.Bytes(conn.Do(redis.HGET, redis.KeyJob, string(jobId)))
		if err == redigo.ErrNil {
			return jobs, werrors.WrapMsgf(err, pkg.ErrJobIdNotFound, string(jobId))
		}
		if err != nil {
			return jobs, werrors.WrapErr(err, pkg.ErrRetrieving)
		}
		err = json.Unmarshal(jobJson, &job)
		if err != nil {
			return jobs, werrors.WrapErr(err, pkg.ErrUnmarshallingJob)
		}
		if owner == job.Owner {
			jobs[i] = job
		}
	}

	return jobs, nil
}

func (rdb RedisDB) Disconnect() error {
	return rdb.redisClient.Disconnect()
}

// getFromHashByJobId is a helper function to search a given hash key for a specified value
func (rdb RedisDB) getFromHashByJobId(conn redigo.Conn, key string, value string) (string, error) {
	result, err := redigo.StringMap(conn.Do(redis.HGETALL, key))
	if err != nil {
		return "", werrors.WrapErr(err, pkg.ErrRetrieving)
	}

	for resKey, resId := range result {
		if resId == value {
			return resKey, nil
		}
	}
	return "", fmt.Errorf(pkg.ErrJobIdNotFound, value)
}

// verifyReply is a helper function to check that each value of the slice is equal to the expected value
func verifyReply(reply []int, expected int) error {
	for _, val := range reply {
		if val != expected {
			return fmt.Errorf("unexpected value in reply, got %d expected %d", val, expected)
		}
	}
	return nil
}

// updateHelper is a recursive function that updates the value in the data passed in using the given keys to index the data
func updateHelper(data *map[string]interface{}, keys []string, value interface{}) error {
	if len(keys) > 1 {
		subMap, ok := (*data)[keys[0]]
		if !ok {
			return fmt.Errorf("entry %s does not exist", strings.Join(keys, "."))
		}
		actualMap, ok := subMap.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected sub structure for key %s", keys[0])
		}
		err := updateHelper(&actualMap, keys[1:], value)
		return err
	}

	_, ok := (*data)[keys[0]]
	if !ok {
		return fmt.Errorf("entry %s does not exist", keys[0])
	}

	(*data)[keys[0]] = value

	return nil
}
