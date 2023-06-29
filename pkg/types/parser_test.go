/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		Name          string
		Parser        map[string]AttributeInfo
		Filename      string
		Attributes    map[string]string
		Expected      map[string]string
		ExpectedError error
	}{
		{
			Name: "Boolean",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "bool"},
				"column":  {Id: "c", DataType: "bool"},
				"channel": {Id: "ch", DataType: "bool"},
			},
			Filename:   "c-ch.tiff",
			Attributes: map[string]string{},
			Expected: map[string]string{
				"row":     "false",
				"column":  "true",
				"channel": "true",
			},
			ExpectedError: nil,
		},
		{
			Name: "Integer",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "int"},
				"column":  {Id: "c", DataType: "int"},
				"channel": {Id: "ch", DataType: "int"},
			},
			Filename:   "r0c2ch003-extra.tiff",
			Attributes: map[string]string{},
			Expected: map[string]string{
				"row":     "0",
				"column":  "2",
				"channel": "3",
			},
			ExpectedError: nil,
		},
		{
			Name: "String",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "string"},
				"column":  {Id: "c", DataType: "string"},
				"channel": {Id: "ch", DataType: "string"},
			},
			Filename:   "r-one-c-two-ch-three.tiff",
			Attributes: map[string]string{},
			Expected: map[string]string{
				"row":     "one",
				"column":  "two",
				"channel": "three",
			},
			ExpectedError: nil,
		},
		{
			Name: "Complex",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "int"},
				"column":  {Id: "c", DataType: "int"},
				"field":   {Id: "f", DataType: "string"},
				"plane":   {Id: "p", DataType: "int"},
				"channel": {Id: "ch", DataType: "string"},
				"isTrue":  {Id: "true", DataType: "bool"},
				"isFalse": {Id: "false", DataType: "bool"},
			},
			Filename:   "r15c007f-test-p0-ch-television-true.tiff",
			Attributes: map[string]string{},
			Expected: map[string]string{
				"row":     "15",
				"column":  "7",
				"field":   "test",
				"plane":   "0",
				"channel": "television",
				"isTrue":  "true",
				"isFalse": "false",
			},
			ExpectedError: nil,
		},
		{
			Name: "No Attributes",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "int"},
				"column":  {Id: "c", DataType: "int"},
				"channel": {Id: "ch", DataType: "int"},
			},
			Filename:      "nothing.tiff",
			Attributes:    map[string]string{},
			Expected:      map[string]string{},
			ExpectedError: nil,
		},
		{
			Name: "Preexisting Attributes No Additions",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "int"},
				"column":  {Id: "c", DataType: "int"},
				"channel": {Id: "ch", DataType: "int"},
			},
			Filename: "nothing.tiff",
			Attributes: map[string]string{
				"Operator": "Bob",
				"Lab":      "Intel",
			},
			Expected: map[string]string{
				"Operator": "Bob",
				"Lab":      "Intel",
			},
			ExpectedError: nil,
		},
		{
			Name: "Preexisting Attributes With Additions",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "int"},
				"column":  {Id: "c", DataType: "int"},
				"channel": {Id: "ch", DataType: "int"},
			},
			Filename: "r8-c5.tiff",
			Attributes: map[string]string{
				"Operator": "Bob",
				"Lab":      "Intel",
			},
			Expected: map[string]string{
				"Operator": "Bob",
				"Lab":      "Intel",
				"row":      "8",
				"column":   "5",
			},
			ExpectedError: nil,
		},
		{
			Name: "Error",
			Parser: map[string]AttributeInfo{
				"row":     {Id: "r", DataType: "sg"},
				"column":  {Id: "c", DataType: "string"},
				"channel": {Id: "ch", DataType: "string"},
			},
			Filename:      "error.tiff",
			Attributes:    map[string]string{},
			Expected:      nil,
			ExpectedError: errors.New("invalid data type for attribute row"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var tempJob Job
			tempJob.InputFile.Name = test.Filename
			tempJob.InputFile.Attributes = test.Attributes

			err := tempJob.InputFile.ParseFilenameForAttributes(test.Parser)
			if test.ExpectedError == nil {
				require.NoError(t, err)
				assert.Equal(t, test.Expected, tempJob.InputFile.Attributes)
			} else {
				assert.Equal(t, test.ExpectedError.Error(), err.Error())
			}
		})
	}
}
