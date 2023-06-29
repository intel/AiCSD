/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import (
	"aicsd/pkg/types"
	"encoding/json"
	"path/filepath"
	"time"
)

// CreateTestTask creates a sample task to use for testing purposes
func CreateTestTask(taskId, description, jobSelector, pipelineId string) types.Task {
	return types.Task{
		Id:               taskId,
		Description:      description,
		JobSelector:      jobSelector,
		PipelineId:       pipelineId,
		ResultFileFolder: filepath.Join(".", "test"),
		ModelParameters:  map[string]string{"Brightness": "0"},
		LastUpdated:      time.Now().UTC().UnixNano(),
	}
}

// CreateJsonTasks creates a slice of json tasks
func CreateJsonTasks(tasks []types.Task) ([]interface{}, error) {

	var jsonTasks []interface{}
	for _, task := range tasks {
		jsonTask, err := json.Marshal(task)
		if err != nil {
			return nil, err
		}

		jsonTasks = append(jsonTasks, jsonTask)
	}

	return jsonTasks, nil
}
