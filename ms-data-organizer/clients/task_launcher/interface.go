/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package task_launcher

import "aicsd/pkg/types"

type Client interface {
	MatchTask(entry types.Job) (bool, error)
	RetrieveById(id string) (types.Task, error)
}
