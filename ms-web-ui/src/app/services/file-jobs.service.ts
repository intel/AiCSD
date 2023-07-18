/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';

import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';

import { Job, Verification } from './data.service';
import { environment } from '../../environments/environment';
import { throwError } from 'rxjs';

@Injectable({
  providedIn: 'root',
})

export class FileJobsService {
  httpOptions = {
    headers: new HttpHeaders({
      'Content-Type': 'application/json; charset=utf-8',
    }),
  };

  constructor(private httpClient: HttpClient) { }

  getAll(): Observable<Job[]> {
    let allJobs = this.httpClient.get<Job[]>(environment.jobApiEndpoint).pipe(
      map(jobList => {
        jobList.forEach(job => {
          // add missing data markers for easier readability
          if (!job.PipelineDetails.QCFlags) {
            job.PipelineDetails.QCFlags = '-';
          }
          if (!job.PipelineDetails.Status) {
            job.PipelineDetails.Status = '-';
          }
          if (!job.PipelineDetails.Results) {
            job.PipelineDetails.Results = '-';
          }
          if (!job.ErrorDetails) {
            job.ErrorDetails.Owner = '-';
            // below is left empty so that only one - is shown on the UI if both fields are empty
            job.ErrorDetails.Error = '';
          }
          if (!job.Verification) {
            job.Verification = Verification.Pending
          }
        });

        // return the modified data:
        return jobList;
     }),
     catchError( error => {
         return throwError(error); // From 'rxjs'
     })
  ); // end of pipe

    return allJobs;
  }

  get(jobId): Observable<Job> {
    return this.httpClient.get<Job>(`${environment.jobApiEndpoint}/${jobId}`);
  }

  addToRejectedDir(jobId: string): Observable<any> {
    let httpOptions = {
      headers: new HttpHeaders({
        'Content-Type': 'application/json; charset=utf-8'
      }),
    };

    return this.httpClient.post<any>(`${environment.rejectApiEndpoint}/${jobId}`, {}, httpOptions)
  }

  removeFromRejectedDir(jobId: string) {
    return this.httpClient.delete<any>(`${environment.rejectApiEndpoint}/${jobId}`)
  }

  updateVerification(jobId: string, verification: Verification) {
    return this.httpClient.put<any>(`${environment.jobApiEndpoint}/${jobId}`, {"Verification": verification})
  }
}
