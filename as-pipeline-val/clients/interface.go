/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package clients

import "aicsd/as-pipeline-val/types"

type Client interface {
	GetPipelines() ([]types.PipelineParams, error)
}
