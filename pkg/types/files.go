/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

import (
	"aicsd/pkg"
	"fmt"
	"os"
	"path/filepath"
)

// FileInfo represents the input file information
type FileInfo struct {
	// Hostname is the host system for the file
	Hostname string
	// DirName is the path to file
	DirName string
	// Name is the file name with its extension
	Name string
	// Extension is just file extension
	Extension string
	// ArchiveName contains the path and name of the archived file
	ArchiveName string
	// Viewable contains the path and name of the viewable file
	Viewable string
	// Attributes contains additional information from configuration/data organizer
	Attributes map[string]string
}

// OutputFile represents the output file after a pipeline has processed
type OutputFile struct {
	// DirName is the path to file
	DirName string
	// Name is the file name with its extension
	Name string
	// Extension is just file extension
	Extension string
	// ArchiveName contains the path and name of the archived file
	ArchiveName string
	// Viewable contains the path and name of the viewable file
	Viewable string
	// Status is the current status of the file
	Status string
	// ErrorDetails is the piece containing user facing error information
	ErrorDetails *pkg.UserFacingError
	// Owner is the component that owns the file as it is processing
	Owner string
}

// CreateOutputFile creates an output file based on the directory, filename, status, errors, and owner passed in.
func CreateOutputFile(dirName, name, extension, archive, view, status, owner string, errDetails *pkg.UserFacingError) OutputFile {
	return OutputFile{
		DirName:      dirName,
		Name:         name,
		Extension:    extension,
		ArchiveName:  archive,
		Viewable:     view,
		Status:       status,
		ErrorDetails: errDetails,
		Owner:        owner,
	}
}

// FullInputFileLocation is a function that will return a string containing information relating to the job's input file.
func (j *Job) FullInputFileLocation() string {
	return fmt.Sprintf("%s:%s", j.InputFile.Hostname, filepath.Join(j.InputFile.DirName, j.InputFile.Name))
}

// FullOutputFileLocation is a function that will return a string containing a list of all the files listed in the
// job.PipelineDetails.OutputFiles field.
func (j *Job) FullOutputFileLocation() string {
	var outputFiles string
	for i, file := range j.PipelineDetails.OutputFiles {
		if i == 0 {
			outputFiles = fmt.Sprintf("%s:%s", j.PipelineDetails.OutputFileHost, file.Name)
		} else {
			outputFiles = fmt.Sprintf("%s, %s", outputFiles, file.Name)
		}
	}
	return outputFiles
}

// UpdateOutputFile is a job method to update the output file fields in the case of error.
func (j *Job) UpdateOutputFile(fileId int, dirName string, name string, ext string, archive string, view string, errDetails error, status, owner string) {
	// don't overwrite current values if pass in empty string
	if dirName != "" {
		j.PipelineDetails.OutputFiles[fileId].DirName = dirName
	}
	if name != "" {
		j.PipelineDetails.OutputFiles[fileId].Name = name
	}
	if ext != "" {
		j.PipelineDetails.OutputFiles[fileId].Extension = ext
	}
	if archive != "" {
		j.PipelineDetails.OutputFiles[fileId].ArchiveName = archive
	}
	if view != "" {
		j.PipelineDetails.OutputFiles[fileId].Viewable = view
	}
	if errDetails != nil {
		j.PipelineDetails.OutputFiles[fileId].ErrorDetails = pkg.CreateUserFacingError(owner, errDetails)
	} else if errDetails == nil {
		j.PipelineDetails.OutputFiles[fileId].ErrorDetails = pkg.CreateUserFacingError("", nil)
	}
	if status != "" {
		j.PipelineDetails.OutputFiles[fileId].Status = status
	}
	if owner != "" {
		j.PipelineDetails.OutputFiles[fileId].Owner = owner
	}
	return
}

// ValidateFiles checks that the files exist on the machine,
// and updates job and file status if otherwise.
func (j *Job) ValidateFiles() error {
	var err error
	for fileId, file := range j.PipelineDetails.OutputFiles {
		file.Owner = j.Owner
		// TODO: implement checksum validation on file
		_, err = os.Stat(filepath.Join(file.DirName, file.Name))
		if err != nil {
			j.Status = pkg.StatusFileError
			j.ErrorDetails = pkg.CreateUserFacingError(j.Owner, pkg.ErrFileInvalid)
			j.UpdateOutputFile(fileId, file.DirName, file.Name, file.Extension, "", "", pkg.ErrFileInvalid, pkg.FileStatusInvalid, j.Owner)
		}
	}

	return err
}
