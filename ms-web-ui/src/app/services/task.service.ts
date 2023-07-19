/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpResponse } from "@angular/common/http";
import { map } from "rxjs/operators";
import { Observable, of } from "rxjs";

import { TaskDetails, OpenAirPipeline, APIResponse, extractData } from './data.service';
import { environment } from '../../environments/environment';
import { Router } from '@angular/router';

@Injectable({
  providedIn: 'root'
})
export class TaskService {
  httpOptions = {
    headers: new HttpHeaders({
      'Content-Type': 'application/json; charset=utf-8'
    }),
    responseType: 'text' as 'json'
  };

  constructor(private httpClient: HttpClient) { }

  delete(id): Observable<any> {
    return this.httpClient.delete(`${environment.taskApiEndpoint}/${id}`);
  }

  getOpenAirPipelines(): Observable<OpenAirPipeline[]> {
    return this.httpClient
      .get<OpenAirPipeline[]>(environment.pipelinesEndpoint)
  }

  add(taskForm): Observable<TaskDetails[]> {
    let newTask = this.createTask(taskForm, null);

    return this.httpClient.post<APIResponse<TaskDetails[]>>(
      environment.taskApiEndpoint, newTask,  this.httpOptions)
       .pipe(map(extractData));
  }

  createTask(taskForm, id): any {
    const taskId = (id == null || id == '') ? null : id;

    const jobSelector = this.buildJobSelectorFromTask(taskForm.JobSelector, taskForm.JobSelectorFile)
    const mapModel = JSON.parse(taskForm.ModelParameters);

    const resultFileFolder = (`${taskForm.ResultFileFolder}`.endsWith('null') || taskForm.ResultFileFolder == "") ? '/tmp/files/output' : `/tmp/files/${taskForm.ResultFileFolder}`

    let task: TaskDetails = {
      Id: taskId,
      Description: taskForm.Description,
      JobSelector: jobSelector,
      PipelineId: taskForm.PipelineId,
      ResultFileFolder: resultFileFolder,
      ModelParameters: mapModel,
      LastUpdated: Math.round(Date.now() * 1000000)
    }

    return task;
  }

  buildJobSelectorFromTask(selectorType: string, filename: string): string {
    const ruleTemplates = {
      matches: `{ "==" : [ { "var" : "InputFile.Name" }, "${filename}" ] }`,
      contains: `{ "in" : [ "${filename}", { "var" : "InputFile.Name" } ] }`
    }

    return ruleTemplates[selectorType];
  }

  getAll(): Observable<TaskDetails[]> {
    return this.httpClient
      .get<TaskDetails[]>(environment.taskApiEndpoint);
  }

  update(taskDetails, id): Observable<any> {
    let updateTask = this.createTask(taskDetails, id);

    return this.httpClient.put(
       environment.taskApiEndpoint, updateTask, this.httpOptions);
  }
}
