/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package redis

import "github.com/gomodule/redigo/redis"

type DBClient interface {
	Disconnect() error
	GetConnection() redis.Conn
	TestConnection() (redis.Conn, error)
}
