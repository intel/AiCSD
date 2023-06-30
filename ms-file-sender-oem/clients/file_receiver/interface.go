/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package file_receiver

import "aicsd/pkg/types"

type Client interface {
	TransmitJob(entry types.Job) error // sends job object and follows up with file(s)
	TransmitFile(id string, entry types.FileInfo) (int, error)
}
