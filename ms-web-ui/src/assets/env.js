/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

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
