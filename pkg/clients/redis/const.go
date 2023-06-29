/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package redis

// Redis commands used in this project
const (
	MULTI   = "MULTI"
	SET     = "SET"
	DEL     = "DEL"
	HSET    = "HSET"
	HGET    = "HGET"
	HGETALL = "HGETALL"
	HKEYS   = "HKEYS"
	HVALS   = "HVALS"
	HEXISTS = "HEXISTS"
	HDEL    = "HDEL"
	EXEC    = "EXEC"
	WATCH   = "WATCH"
)

const (
	DBKeySeparator = ":"
	KeyJob         = "job"
	KeyInputFile   = "job|input_file"
	KeyLock        = "lock"
	KeyOwner       = "job|owner"
	KeyTask        = "task"
)
