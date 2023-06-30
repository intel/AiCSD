/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package werrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aicsd/pkg"
)

func TestWrapErr(t *testing.T) {
	causeErr := errors.New("this caused the error")
	errContext := pkg.ErrHandleJob
	expected := fmt.Sprintf("%s: %s", errContext.Error(), causeErr.Error())

	wrappedWithErr := WrapErr(causeErr, errContext)

	actual := wrappedWithErr.Error()
	assert.Equal(t, expected, actual)
}

func TestWrapMsg(t *testing.T) {
	causeErr := errors.New("this caused the error")
	errContext := pkg.ErrHandleJob.Error()
	expected := fmt.Sprintf("%s: %s", errContext, causeErr.Error())

	wrappedWithMsg := WrapMsg(causeErr, errContext)

	actual := wrappedWithMsg.Error()
	assert.Equal(t, expected, actual)
}

func TestWrapMsgf(t *testing.T) {
	causeErr := errors.New("this caused the error")
	errContextFormat := "this is %s for the error"
	expected := fmt.Sprintf("%s: %s", fmt.Sprintf(errContextFormat, "formatted context"), causeErr.Error())

	wrappedWithMsgf := WrapMsgf(causeErr, errContextFormat, "formatted context")

	actual := wrappedWithMsgf.Error()
	assert.Equal(t, expected, actual)
}

func TestCause(t *testing.T) {
	expected := errors.New("this caused the error")
	errContext := pkg.ErrHandleJob

	wrappedWithErr := WrapErr(expected, errContext)

	actual := Cause(wrappedWithErr)
	require.NotNil(t, actual)
	assert.ErrorIs(t, expected, actual)
}

func TestContext(t *testing.T) {
	causeErr := errors.New("this caused the error")
	expected := pkg.ErrHandleJob

	wrappedWithErr := WrapErr(causeErr, expected)

	actual := Context(wrappedWithErr)
	require.NotNil(t, actual)
	assert.ErrorIs(t, expected, actual)
}

func TestContextMsg(t *testing.T) {
	causeErr := errors.New("this caused the error")
	expected := pkg.ErrHandleJob.Error()

	wrappedWithMsg := WrapMsg(causeErr, expected)

	actual := ContextMsg(wrappedWithMsg)
	assert.Equal(t, expected, actual)
}
