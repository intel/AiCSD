/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package pipeline_sim_tests

import (
	integrationtests "aicsd/integration-tests/pkg"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func changeTaskObj(taskObj *types.Task) {
	taskObj.Description = "Generate Output File V2"
	taskObj.JobSelector = "{ \"in\" : [ \"test-image\", {\"var\" : \"InputFile.Name\" } ] }"
	taskObj.PipelineId = "New Pipleline ID"

}

func TestCRUDTaskRepo(t *testing.T) {

	taskObj := helpers.CreateTestTask("test.tiff", "only-file")

	// create httpexpect instance
	e := httpexpect.Default(t, integrationtests.TaskLauncherUrl)

	//Perform GET When No POST has been performed
	integrationtests.GET_Task(e, &taskObj, http.StatusNoContent, true)

	//POST instance
	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)

	//Make a POST for the same taskObj
	integrationtests.POST_Task(e, &taskObj, http.StatusBadRequest, true)

	//Get http instance
	integrationtests.GET_Task(e, &taskObj, http.StatusOK, false)

	//Modify task Object to test PUT
	changeTaskObj(&taskObj)

	//Update http instance
	integrationtests.PUT_Task(e, &taskObj, http.StatusOK, false)

	//Delete instance
	integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	//Check if updating is possible after deletion
	integrationtests.PUT_Task(e, &taskObj, http.StatusNotFound, true)

	//Check if deleting twice is possible
	integrationtests.DELETE_Task(e, &taskObj, http.StatusBadRequest, true)

	//Check Get after deletion
	integrationtests.GET_Task(e, &taskObj, http.StatusNoContent, true)

	//Do 2 separate POSTS, 1 GET, 1 DELETE then another GET
	taskObj.Id = ""
	taskObj2 := taskObj

	taskObj.Id = integrationtests.POST_Task(e, &taskObj, http.StatusCreated, false)
	taskObj2.Id = integrationtests.POST_Task(e, &taskObj2, http.StatusCreated, false)

	integrationtests.GET_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.DELETE_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.GET_Task(e, &taskObj, http.StatusOK, false)

	integrationtests.DELETE_Task(e, &taskObj2, http.StatusOK, false)

	integrationtests.GET_Task(e, &taskObj, http.StatusNoContent, true)

}
