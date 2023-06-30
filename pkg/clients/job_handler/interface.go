/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package job_handler

import "aicsd/pkg/types"

type Client interface {
	HandleJob(entry types.Job) error
}
