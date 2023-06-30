/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package job_repo

import (
	"aicsd/pkg/types"
)

type Client interface {
	Create(job types.Job) (string, bool, error)
	RetrieveAll(headers map[string]string) ([]types.Job, error)
	RetrieveAllByOwner(owner string) ([]types.Job, error)
	RetrieveById(id string) (types.Job, error)
	Update(id string, jobFields map[string]interface{}) (types.Job, error)
	Delete(id string) error
}
