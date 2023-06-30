/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package redis

import (
	"aicsd/pkg/werrors"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/hashicorp/go-multierror"
)

type Client struct {
	pool *redis.Pool
}

// NewClient returns a pointer to a redis client
func NewClient(host string, port string, timeout time.Duration, secrets map[string]string) DBClient {
	connectionString := fmt.Sprintf("%s:%s", host, port)
	var username, password string
	var ok bool
	username, ok = secrets["username"]
	if !ok {
		username = ""
	}

	password, ok = secrets["password"]
	if !ok {
		password = ""
	}

	opts := []redis.DialOption{
		redis.DialConnectTimeout(timeout),
		redis.DialUsername(username),
		redis.DialPassword(password),
	}

	// this function is called the first time the connection is needed
	dialFunc := func() (redis.Conn, error) {
		conn, err := redis.Dial("tcp", connectionString, opts...)
		if err != nil {
			return nil, fmt.Errorf("could not dial Redis: %s", err)
		}

		_, err = conn.Do("PING")
		if err != nil {
			return nil, fmt.Errorf("could not ping Redis: %s", err)
		}
		return conn, nil
	}

	return &Client{
		pool: &redis.Pool{
			IdleTimeout: timeout,
			Dial:        dialFunc,
		},
	}
}

// Disconnect ends the connection
func (c *Client) Disconnect() error {
	return c.pool.Close()
}

// GetConnection returns a connection from the redis pool
// any calls to GetConnection will need to have a subsequent defer func() { _ = conn.Close() }()
func (c *Client) GetConnection() redis.Conn {
	return c.pool.Get()
}

// TestConnection returns a connection from the redis pool
func (c *Client) TestConnection() (redis.Conn, error) {
	conn, err := c.pool.Dial()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// CreateKey concatenates all targets passed in with the database key separator
func CreateKey(targets ...string) string {
	return strings.Join(targets, DBKeySeparator)
}

// HashDeleteByKey gets all entries for a hash key and deletes each one
func HashDeleteByKey(conn redis.Conn, key string) error {
	var errs error
	hash, err := redis.StringMap(conn.Do(HGETALL, key))
	if err != nil && err != redis.ErrNil {
		return werrors.WrapMsgf(err, "could not get all from hash by key %s", key)
	} else if err == redis.ErrNil {
		return fmt.Errorf("no results for key %s", key)
	}
	for field := range hash {
		num, err := redis.Int(conn.Do(HDEL, key, field))
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if num == 0 {
			errs = multierror.Append(errs, fmt.Errorf("no value to delete for key = %s, field = %s", key, field))
		}
	}
	return errs
}
