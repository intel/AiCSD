package pkg

import (
	"aicsd/pkg"
	"aicsd/pkg/types"
	"github.com/gavv/httpexpect/v2"
)

// The *_Task functions have an isNegativeTest bool parameter that is used to determine if we retry POST PUT etc 10 times (MaxRetries value)
func POST_Task(e *httpexpect.Expect, taskObj *types.Task, status int, isNegativeTest bool) string {
	var ID *httpexpect.Response
	if !isNegativeTest {
		ID = e.POST(pkg.EndpointTask).WithJSON(taskObj).
			WithMaxRetries(MaxRetries).
			WithRetryPolicy(httpexpect.RetryAllErrors).
			Expect().
			Status(status)
	} else {
		ID = e.POST(pkg.EndpointTask).WithJSON(taskObj).
			Expect().
			Status(status)
	}

	return ID.Text().Raw()
}

func GET_Task(e *httpexpect.Expect, taskObj *types.Task, status int, isNegativeTest bool) {
	if !isNegativeTest {
		e.GET(pkg.EndpointTask).WithJSON(taskObj).
			WithMaxRetries(MaxRetries).
			WithRetryPolicy(httpexpect.RetryAllErrors).
			Expect().
			Status(status)
	} else {
		e.GET(pkg.EndpointTask).WithJSON(taskObj).
			Expect().
			Status(status)
	}
}

func PUT_Task(e *httpexpect.Expect, taskObj *types.Task, status int, isNegativeTest bool) {
	if !isNegativeTest {
		e.PUT(pkg.EndpointTask).WithJSON(taskObj).
			WithMaxRetries(MaxRetries).
			WithRetryPolicy(httpexpect.RetryAllErrors).
			Expect().
			Status(status)
	} else {
		e.PUT(pkg.EndpointTask).WithJSON(taskObj).
			Expect().
			Status(status)
	}

}

func DELETE_Task(e *httpexpect.Expect, taskObj *types.Task, status int, isNegativeTest bool) {
	if !isNegativeTest {
		e.DELETE(pkg.EndpointTask + "/" + taskObj.Id).WithJSON(taskObj).
			WithMaxRetries(MaxRetries).
			WithRetryPolicy(httpexpect.RetryAllErrors).
			Expect().
			Status(status)
	} else {
		e.DELETE(pkg.EndpointTask + "/" + taskObj.Id).WithJSON(taskObj).
			Expect().
			Status(status)
	}

}
