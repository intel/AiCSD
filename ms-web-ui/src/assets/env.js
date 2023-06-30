// INTEL CONFIDENTIAL

// Copyright (C) 2023 Intel Corporation

// This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
// license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
// copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

// This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.

// This file can be replaced during build by using the `fileReplacements` array.
// `ng build --prod` replaces `environment.ts` with `environment.prod.ts`.
// The list of file replacements can be found in `angular.json`.

(function(window) {
    window["env"] = window["env"] || {};

    //Environmental variable
    window["env"]["jobApiUrl"] = "http://localhost:59784/api/v1/job";
    window["env"]["taskApiUrl"] = "http://localhost:59785/api/v1/task";
    window["env"]["pipelinesApiUrl"] = "http://localhost:10107/api/v1/pipelines";
    window["env"]["ModelApiUrl"] = "http://localhost:8080/upload";
    window["env"]["rejectApiUrl"] = "http://localhost:59786/api/v1/reject";
})(this);
