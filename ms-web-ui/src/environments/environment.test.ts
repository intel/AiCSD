// INTEL CONFIDENTIAL

// Copyright (C) 2023 Intel Corporation

// This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
// license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
// copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

// This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.

export const environment = {
    production: false,
    apiServer: '/api/v1',
    jobApiEndpoint: "http://localhost:59784/api/v1/job",
    taskApiEndpoint: "http://localhost:59785/api/v1/task",
    pipelinesEndpoint: "http://localhost:10107/api/v1/pipelines",
    modelApiEndpoint: "http://localhost:8080/upload",
    rejectApiEndpoint: "http://localhost:59786/api/v1/reject"
  };
