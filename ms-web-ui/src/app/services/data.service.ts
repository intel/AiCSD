/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

import { Injectable } from '@angular/core';

/**
 * Represents a page that users can navigate to, either via
 * a URL route, the navbar, or from other component interactions.
 *
 * @interface
 */
export interface Page {
  page: 'jobs' | 'task' | 'grafana' | 'upload';
  caption: string;
}

/**
 * Represents Task Details
 *
 * @interface
 */
export interface TaskDetails {
  Id: string;
  Description: string;
  JobSelector: string | undefined;
  PipelineId: string | undefined;
  ResultFileFolder: string | undefined;
  ModelParameters: Map<string,string> | undefined;
  LastUpdated: number | undefined;
}

/**
 * Represents Model Details
 *
 * @interface
 */
export interface Model {
  Name: string;
  Type: string;
  Zip: File;
}

/**
 * Represents ErrorDetails for jobs and files
 *
 * @interface
 */
export interface UserFacingError {
  Owner: string;
  Error: string;
}

export enum Verification {
  Pending,
  Accepted,
  Rejected
}

 /**
  * Represents Job details
  *
  * @interface
  */
export interface Job {
  Id: string;
  Owner: string;
  Status: string;
  InputFile: InputFile;
  PipelineDetails: PipelineDetails;
  ErrorDetails: UserFacingError | undefined;
  Verification: Verification | Verification.Pending;
}

/**
 * Represents File details
 *
 * @interface
 */
export interface InputFile {
  Hostname: string;
  DirName: string;
  Name: string;
  ArchiveName: string;
  Viewable: string;
  Extension: string;
  Attributes: {[name: string]: string};
}

/**
 * Represents Pipeline details
 *
 * @interface
 */
export interface PipelineDetails {
  TaskId: string;
  Status: string;
  QCFlags: string;
  OutputFileHost: string;
  OutputFiles: OutputFile[];
  Results: string;
}

export interface OpenAirPipeline {
  Id: string;
  Name: string;
  Description: string;
  SubscriptionTopic: string;
  Status: string;
}

export interface OutputFile {
  DirName: string;
  Name: string;
  Extension: string;
  ArchiveName: string;
  Viewable: string;
  Status: string;
  ErrorDetails: UserFacingError | undefined;
  Owner: string;
}

export interface APIResponse<T> {
  data: T | undefined;
  error: string | undefined;
}

export function extractData<T>(r: APIResponse<T>): T {
  if (!r) {
    return;
  }

  if (r.error) {
    throw r.error;
  }

  return r.data;
}

@Injectable({
  providedIn: 'root',
})
export class DataService {
  constructor() {}

  // pages is a list of all tabs that are navigable by the user, accessible
  // via the routing module and also by clicking tabs
  public pages: Page[] = [
    { page: 'task', caption: 'Create/Modify Tasks' },
    { page: 'jobs', caption: 'View Jobs' },
    { page: 'grafana', caption: 'Dashboards' },
    { page: 'upload', caption: 'Upload Models' },
  ];

  // currentPage is set whenever the user taps/clicks on a tab or navigates
  // to a page via the routing module. Its value is important because it is
  // visually bound to highlighting the currently viewed tab
  public currentPage: 'task' | 'jobs' | 'grafana' | 'upload';
}
