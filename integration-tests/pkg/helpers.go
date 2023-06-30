/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import (
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/translation"
	"aicsd/pkg/types"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

// CopyFile is a helper function to take a sourceFile and copy it to a destinationFile.
func CopyFile(t *testing.T, sourceFile string, destinationFile string) {
	t.Helper()
	contents, err := os.ReadFile(sourceFile)
	require.NoError(t, err)
	require.NotNil(t, contents)
	err = os.WriteFile(destinationFile, contents, pkg.FilePermissions)
	require.NoError(t, err)
}

// MatchOwner is a helper function to check that the owner matches the string input and returns the job from the job repo
func MatchOwner(t *testing.T, jobRepoClient job_repo.Client, jobId string, owner string) types.Job {
	t.Helper()
	job, err := jobRepoClient.RetrieveById(jobId)
	require.NoError(t, err)
	require.Equal(t, owner, job.Owner)
	return job
}

// ParseServiceLog queries the url for a docker log and searches it for a given substring. It returns once the log
// has been searched. Returns true if the substring is found and false if the substring is not found
func ParseServiceLog(t *testing.T, url string, subStr string) bool {
	t.Helper()
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := net.Dial("unix", "/var/run/docker.sock")
				if err != nil {
					return nil, fmt.Errorf("cannot connect docker socket: %v", err)
				}
				return conn, nil
			}},
		Timeout: 1 * time.Minute,
	}
	resp, err := client.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), subStr) {
			return true
		}
	}
	return false
}

// Contains is a helper function to check if a slice contains a given string
func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if strings.Contains(s, str) {
			return true
		}
	}
	return false
}

// NewTestClient is a helper function to create a httpClient
func NewTestClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 15,
	}
}

// AssertInternationalization is a helper function to verify that when the translation header is set to Chinese,
// then the job fields that can be translated are translated properly.
func AssertInternationalization(t *testing.T, jobRepoClient job_repo.Client, jobStatus, fileStatus, jobOwner, fileOwner string,
	jobErrorDetails, fileErrorDetails *pkg.UserFacingError) {
	t.Helper()
	var (
		localizationFiles = []string{"../../pkg/translation/dictionary/en.json", "../../pkg/translation/dictionary/zh.json"}
	)

	// make GET request to job repo using appropriate header to translate job fields using internal app logic.
	translatedJobs, err := jobRepoClient.RetrieveAll(map[string]string{pkg.AcceptLanguage: pkg.LanguageChinese})
	require.NoError(t, err)
	assert.NotNil(t, translatedJobs)

	// create test bundler to verify resulting job fields were translated
	testLocalizationBundle, err := translation.NewBundler(localizationFiles)
	require.NoError(t, err)
	loc := i18n.NewLocalizer(testLocalizationBundle, pkg.LanguageChinese)

	// check translated job status
	translatedJobStatus, err := translation.TranslateField(loc, jobStatus)
	assert.NoError(t, err)
	assert.Equal(t, translatedJobStatus, translatedJobs[0].Status)

	// check translated job owner
	translatedJobOwner, err := translation.TranslateField(loc, jobOwner)
	assert.NoError(t, err)
	assert.Equal(t, translatedJobOwner, translatedJobs[0].Owner)

	// check translated job error details
	translatedJobErrorDetails, err := translation.TranslateErrorDetails(loc, jobErrorDetails)
	assert.NoError(t, err)
	assert.Equal(t, translatedJobErrorDetails, translatedJobs[0].ErrorDetails)

	// check translated file status
	translatedFileStatus, err := translation.TranslateField(loc, fileStatus)
	assert.NoError(t, err)
	assert.Equal(t, translatedFileStatus, translatedJobs[0].PipelineDetails.OutputFiles[0].Status)

	// check translated file error details
	translatedFileErrorDetails, err := translation.TranslateErrorDetails(loc, fileErrorDetails)
	assert.NoError(t, err)
	assert.Equal(t, translatedFileErrorDetails, translatedJobs[0].PipelineDetails.OutputFiles[0].ErrorDetails)

	// check translated file owner
	translatedFileOwner, err := translation.TranslateField(loc, fileOwner)
	assert.NoError(t, err)
	assert.Equal(t, translatedFileOwner, translatedJobs[0].PipelineDetails.OutputFiles[0].Owner)
}

// IsServiceReady is a helper function to determine if the endpoint for a service is pingable
func IsServiceReady(url string) bool {
	resp, err := http.Get(url)
	return (resp != nil && err == nil && resp.StatusCode == http.StatusOK)
}

// IsServiceDown is a helper to determine of a service is unavailable
func IsServiceDown(serviceUrl string) bool {
	_, get_err := http.Get(serviceUrl)
	return (get_err != nil)
}

// AreAllServicesDown is a helper to determine if all services are down within the allotted PauseTime
func AreAllServicesDown() bool {
	deadline := time.Now().Add(PauseTime)
	for time.Now().Before(deadline) {
		if IsServiceDown(JobRepositoryUrl+PingEndpoint) && IsServiceDown(FileWatcherUrl+PingEndpoint) &&
			IsServiceDown(DataOraganizerUrl+PingEndpoint) && IsServiceDown(TaskLauncherUrl+PingEndpoint) &&
			IsServiceDown(ConsulUrl+PingEndpoint) && IsServiceDown(RedisUrl+PingEndpoint) {
			return true
		} else {
			time.Sleep(1 * time.Second)
		}
	}
	return false
}

func CallRetry(t *testing.T, serviceUrl string) {
	t.Helper()
	var err error
	var r *http.Response
	retryUrl := fmt.Sprintf("%s%s", serviceUrl, pkg.EndpointRetry)
	require.Eventually(t, func() bool {
		switch serviceUrl {
		case TaskLauncherUrl:
			r, err = http.Post(retryUrl, "application/json", bytes.NewReader([]byte("{ \"TimeoutDuration\":\"1s\" }")))
		default:
			r, err = http.Post(retryUrl, "", nil)
		}
		require.NoError(t, err)
		return r.StatusCode == http.StatusOK
	}, PauseTime, time.Second)
}
