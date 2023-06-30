/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

import (
	"fmt"
	"regexp"
	"strconv"
)

// FilenameDecoder is a wrapper for loading attribute parser from toml
type FilenameDecoder struct {
	AttributeParser map[string]AttributeInfo
}

// AttributeInfo defines how to parse a single file attribute of FileInfo
type AttributeInfo struct {
	Id       string
	DataType string
}

// ParseFilenameForAttributes takes in a parsing structure and parses the Name stored in FileInfo into the Attributes in FileInfo
func (fi *FileInfo) ParseFilenameForAttributes(parser map[string]AttributeInfo) error {

	attributes := make(map[string]string)
	const integer string = `\d+`
	const word string = `\W[a-z]+`
	const nonLetter string = `[^a-z]`

	regInt, err := regexp.Compile(integer)
	if err != nil {
		return err
	}
	regString, err := regexp.Compile(word)
	if err != nil {
		return err
	}

	for attribute, data := range parser {
		switch data.DataType {
		// If the id is found and is not contained within a larger id then the attribute is true
		case "bool":
			// "-" picks up an edge case where the bool is at the start of the filename
			match, err := regexp.MatchString(nonLetter+data.Id+nonLetter, "-"+fi.Name)
			if err != nil {
				return err
			}
			if match {
				attributes[attribute] = "true"
			} else {
				attributes[attribute] = "false"
			}
		// Looks for pairs of attribute id's and integers for example t0.tiff id = t, integer = 0
		case "int":
			regAttribute, err := regexp.Compile(data.Id + integer)
			if err != nil {
				return err
			}
			if number := regInt.FindString(regAttribute.FindString(fi.Name)); number != "" {
				numberInt, err := strconv.Atoi(number)
				if err != nil {
					return fmt.Errorf("DataType int does not match data")
				}
				attributes[attribute] = strconv.Itoa(numberInt)
				if attributes[attribute] == "" {
					attributes[attribute] = "0"
				}
			}
		// Looks for pairs of id's and strings seperated by any nonLetter character
		case "string":
			regAttribute, err := regexp.Compile(data.Id + word)
			if err != nil {
				return err
			}
			temp := regString.FindString(regAttribute.FindString(fi.Name))
			if len(temp) > 1 {
				attributes[attribute] = temp[1:]
			}
		default:
			return fmt.Errorf("invalid data type for attribute %s", attribute)
		}
	}

	// Add attributes to job
	for k, v := range attributes {
		fi.Attributes[k] = v
	}

	return nil
}

// UpdateFromRaw defines how the FilenameDecoder will be loaded when service.LoadCustomConfig is called
func (d *FilenameDecoder) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*FilenameDecoder)
	if !ok {
		return false
	}

	*d = *configuration

	return true
}
