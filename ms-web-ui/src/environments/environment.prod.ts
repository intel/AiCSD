/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

export const environment = {
  production: true,
  apiServer: '/api/v1',
  jobApiEndpoint: window["env"]["jobApiUrl"], 
  taskApiEndpoint: window["env"]["taskApiUrl"],
  pipelinesEndpoint: window["env"]["pipelinesApiUrl"],
  modelApiEndpoint: window["env"]["ModelApiUrl"],
  rejectApiEndpoint: window["env"]["rejectApiUrl"],
};
