// INTEL CONFIDENTIAL

// Copyright (C) 2023 Intel Corporation

// This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
// license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
// copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

// This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.

// This file is required by karma.conf.js and loads recursively all the .spec and framework files

import 'zone.js/dist/zone-testing';
import { getTestBed } from '@angular/core/testing';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting
} from '@angular/platform-browser-dynamic/testing';

// First, initialize the Angular testing environment.
getTestBed().initTestEnvironment(
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting(), {
    teardown: { destroyAfterEach: false }
}
);
