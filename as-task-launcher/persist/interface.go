/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package persist

import "aicsd/pkg/types"

const (
	StatusCreated = "Created"
	StatusExists  = "Exists"
)

type Persistence interface {
	Create(task types.Task) (string, error)
	Update(task types.Task) error
	Delete(id string) error
	GetById(id string) (task types.Task, err error)
	GetAll() ([]types.Task, error)
	Filter(task types.Task) (results []types.Task, err error)
	Disconnect() error
}
