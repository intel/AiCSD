/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package auth

import "net/http"

type Jwt interface {
	createToken() (string, error)
	AddAuthHeader(req *http.Request) error
}
