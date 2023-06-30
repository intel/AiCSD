/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import "fmt"

// Common task errors
var (
	ErrDuplicateTaskId           = fmt.Errorf("id already specified")
	ErrMarshallingTask           = fmt.Errorf("failed to marshal tasks")
	ErrUnmarshallingTask         = fmt.Errorf("failed to unmarshal tasks")
	ErrTaskCreation              = fmt.Errorf("failed to create task")
	ErrTaskRetrieval             = fmt.Errorf("failed to retrieve task(s)")
	ErrTaskEmptyJobSelector      = fmt.Errorf("job selector field is empty")
	ErrTaskEmptyPipelineId       = fmt.Errorf("pipeline id field is empty")
	ErrTaskEmptyDescription      = fmt.Errorf("description field is empty")
	ErrReqBodyTaskStatusMismatch = fmt.Errorf("request body does not match expected task status")
	ErrFmtTaskIdNotFound         = "specified task id %s not found"
	ErrDeleteTask                = fmt.Errorf("failed to delete task")
	ErrTaskEmpty                 = fmt.Errorf("no task is returned")
	ErrTaskIdEmpty               = fmt.Errorf("no task id is specified")
	ErrNoTaskDeleted             = fmt.Errorf("no task was deleted")
	ErrTaskInvalid               = fmt.Errorf("failed to validate task")
	ErrTaskRetrieving            = fmt.Errorf("failed to retrieve task from task launcher")
)
