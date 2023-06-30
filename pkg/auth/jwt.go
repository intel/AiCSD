/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package auth

import (
	"aicsd/pkg/werrors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// set the default token expiration to 5 mins
const (
	defaultTokenExpiration = 60 * 5 * time.Second
	AlgES256               = "ES256"
	AlgRS256               = "RS256"
)

type JWTInfo struct {
	algorithm      string
	privateKeyPath string
	jwtKey         string
	expiration     time.Duration
}

func NewToken(algorithm string, privatePath string, jwtKeyPath string, expiration string) (*JWTInfo, error) {
	if algorithm != AlgES256 && algorithm != AlgRS256 {
		return nil, fmt.Errorf("Unsupported jwt algorithm, got %s but must be %s or %s", algorithm, AlgES256, AlgRS256)
	}
	bytes, err := os.ReadFile(jwtKeyPath)
	if err != nil {
		return nil, werrors.WrapMsgf(err, "Could not parse JWT Key file %s", jwtKeyPath)
	}
	token := JWTInfo{
		algorithm:      algorithm,
		privateKeyPath: privatePath,
		jwtKey:         strings.TrimSpace(string(bytes)),
	}
	if len(expiration) > 0 {
		duration, err := time.ParseDuration(expiration)
		if err != nil {
			return nil, werrors.WrapMsg(err, "Could not parse JWT duration")
		}
		token.expiration = duration
	} else {
		token.expiration = defaultTokenExpiration
	}
	return &token, nil
}

// createToken generates a jwt token to be used in https calls using one of the supported algorithm types
func (j *JWTInfo) createToken() (string, error) {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		Issuer:    j.jwtKey,
		ExpiresAt: jwt.NewNumericDate(now.Add(j.expiration)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	bytes, err := os.ReadFile(j.privateKeyPath)
	if err != nil {
		return "", werrors.WrapMsgf(err, "Could not read private key from %s", j.privateKeyPath)
	}
	var signingMethod jwt.SigningMethod
	var key interface{}
	switch j.algorithm {
	case AlgES256:
		signingMethod = jwt.SigningMethodES256
		ecKey, err := jwt.ParseECPrivateKeyFromPEM(bytes)
		if err == nil && ecKey.Params().BitSize != 256 {
			return "", werrors.WrapMsgf(err, "Key bit size is incorrect (%d instead of 256)", ecKey.Params().BitSize)
		}
		key = ecKey
	case AlgRS256:
		signingMethod = jwt.SigningMethodRS256
		key, err = jwt.ParseRSAPrivateKeyFromPEM(bytes)
	default:
		return "", fmt.Errorf("Unsupported jwt algorithm, got %s but must be %s or %s", j.algorithm, AlgES256, AlgRS256)
	}
	if err != nil {
		return "", werrors.WrapMsgf(err, "Could not parse %s private key from %s", j.algorithm, j.privateKeyPath)
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", werrors.WrapMsg(err, "Could not create the signed JWT token")
	}
	return signedToken, nil
}

// AddAuthHeader creates a token to add to the request header if the jwt information is set
func (j *JWTInfo) AddAuthHeader(req *http.Request) error {
	if j != nil {
		signingToken, err := j.createToken()
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", signingToken))
	}
	return nil
}
