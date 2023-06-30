/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

import "time"

// Task Object Attributes
type Task struct {
	// Id is the unique identifier for task
	Id string
	// Description describes what the task does
	Description string
	// JobSelector contains a filter to select the Job
	JobSelector string
	// PipelineId is the unique identifier for each pipeline
	PipelineId string
	// ResultFileFolder is the path to inference result file (CSV or Image files)
	ResultFileFolder string
	// ModelParameters are parameters specific to a pipeline and applied as data is launched in pipeline execution
	ModelParameters map[string]string
	// LastUpdated is the update time in ns from UTC
	LastUpdated int64
}

// ReplaceTask will replace the values from the existing task object from redisdb with the incoming task object
func (t *Task) ReplaceTask(task Task) {
	if task.Id != "" {
		t.Id = task.Id
	}
	if task.Description != "" {
		t.Description = task.Description
	}
	if task.JobSelector != "" {
		t.JobSelector = task.JobSelector
	}
	if task.PipelineId != "" {
		t.PipelineId = task.PipelineId
	}
	if task.ResultFileFolder != "" {
		t.ResultFileFolder = task.ResultFileFolder
	}
	if len(task.ModelParameters) != 0 {
		for id, val := range task.ModelParameters {
			t.ModelParameters[id] = val
		}
	}
	t.SetLastUpdated()
}

func (t *Task) SetLastUpdated() {
	t.LastUpdated = time.Now().UTC().UnixNano()
}
