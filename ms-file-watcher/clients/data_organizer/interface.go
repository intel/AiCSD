/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package data_organizer

type Client interface {
	NotifyNewFile(filename string) error
}
