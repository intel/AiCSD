/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

// Note: The inspiration for this code came from https://github.com/pkg/errors which has been archived and is no longer
// maintained. We are unable to use https://github.com/pkg/errors directly since it is no longer being maintained.
// Much of this code is very much like that in https://github.com/pkg/errors/blob/master/errors.go.
// Enhancements have been added as well as some name changes. We chose to use "werrors" as the package name to avoid
// conflicts with the Go "errors" package.
//

package werrors

import "fmt"

type wrappedWithError struct {
	cause   error
	context error
}

type wrappedWithMessage struct {
	cause   error
	context string
}

// Error returns the combined error message from the cause error and context error
func (w *wrappedWithError) Error() string {
	return fmt.Sprintf("%s: %s", w.context.Error(), w.cause.Error())
}

// Cause returns the cause error from the wrapped error
func (w *wrappedWithError) Cause() error {
	return w.cause
}

// Context returns the context error from the wrapped error
func (w *wrappedWithError) Context() error {
	return w.context
}

// Error returns the combined error message from the cause error and context message
func (w *wrappedWithMessage) Error() string {
	return fmt.Sprintf("%s: %s", w.context, w.cause.Error())
}

// Cause returns the cause error from the wrapped error
func (w *wrappedWithMessage) Cause() error {
	return w.cause
}

// Context returns the context message from the wrapped error
func (w *wrappedWithMessage) Context() string {
	return w.context
}

// WrapErr wraps a cause error with an error giving context
func WrapErr(cause error, context error) error {
	return &wrappedWithError{
		cause:   cause,
		context: context,
	}
}

// WrapMsg wraps a cause error with a message giving context
func WrapMsg(cause error, context string) error {
	return &wrappedWithMessage{
		cause:   cause,
		context: context,
	}
}

// WrapMsgf wraps a cause error with a formatted message giving context
func WrapMsgf(cause error, format string, args ...interface{}) error {
	return &wrappedWithMessage{
		cause:   cause,
		context: fmt.Sprintf(format, args...),
	}
}

// Cause returns the cause error from a wrapped error.
// nil will be returned if the error is not a Wrapped error
// Useful in unit tests when comparing result error object against expect error object that cause error
func Cause(err error) error {

	type causer interface {
		Cause() error
	}

	if err == nil {
		return nil
	}

	causeErr, ok := err.(causer)
	if !ok {
		return nil
	}

	return causeErr.Cause()
}

// Context returns the context error from a wrapped error.
// nil will be returned if the error is not a Wrapped error
// Useful in unit tests when comparing result error object against expect error object that is the context error
func Context(err error) error {
	type errContext interface {
		Context() error
	}

	if err == nil {
		return nil
	}

	contextErr, ok := err.(errContext)
	if !ok {
		return nil
	}

	return contextErr.Context()
}

// ContextMsg returns the context message from a wrapped error.
// Empty string will be returned if the error is not a Wrapped error
// Useful in unit tests when comparing result error message against expect error message
func ContextMsg(err error) string {
	type errContext interface {
		Context() string
	}

	if err == nil {
		return ""
	}

	contextMsg, ok := err.(errContext)
	if !ok {
		return ""
	}

	return contextMsg.Context()
}
