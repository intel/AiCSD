/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import (
	"aicsd/pkg/types"
)

const (
	StatusCreated = "Created"
	StatusExists  = "Exists"
	StatusNone    = ""
)

type Persistence interface {
	Create(job types.Job) (string, types.Job, error)
	Update(id string, values map[string]interface{}) (types.Job, error)
	Delete(id string) error
	GetAll() ([]types.Job, error)
	GetById(id string) (types.Job, error)
	GetByOwner(owner string) ([]types.Job, error)
	Disconnect() error
}
