/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package file_sender

type Client interface {
	TransmitFile(jobId, fileId string) ([]byte, error)
	ArchiveFile(jobId string) error
	Retry() error
}
