/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

// This file can be replaced during build by using the `fileReplacements` array.
// `ng build --prod` replaces `environment.ts` with `environment.prod.ts`.
// The list of file replacements can be found in `angular.json`.

(function(window){
    window["env"] = window["env"] || {};

    //Environmental variable
    window["env"]["jobApiUrl"] = "${JOB_API_URL}";
    window["env"]["taskApiUrl"] = "${TASK_API_URL}";
    window["env"]["pipelinesApiUrl"] = "${PIPELINES_API_URL}";
    window["env"]["ModelApiUrl"] = "${MODEL_API_URL}";
    window["env"]["rejectApiUrl"] = "${REJECT_API_URL}";
})(this);