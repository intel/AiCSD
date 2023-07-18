/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

// This file can be replaced during build by using the `fileReplacements` array.
// `ng build --prod` replaces `environment.ts` with `environment.prod.ts`.
// The list of file replacements can be found in `angular.json`.

export const environment = {
  production: false,
  apiServer: '/api/v1',
  jobApiEndpoint: window["env"]["jobApiUrl"], 
  taskApiEndpoint: window["env"]["taskApiUrl"],
  pipelinesEndpoint: window["env"]["pipelinesApiUrl"],
  modelApiEndpoint: window["env"]["ModelApiUrl"],
  rejectApiEndpoint: window["env"]["rejectApiUrl"],
};

/*
 * For easier debugging in development mode, you can import the following file
 * to ignore zone related error stack frames such as `zone.run`, `zoneDelegate.invokeTask`.
 *
 * This import should be commented out in production mode because it will have a negative impact
 * on performance if an error is thrown.
 */
// import 'zone.js/dist/zone-error';  // Included with Angular CLI.
