/* INTEL CONFIDENTIAL

 Copyright (C) 2023 Intel Corporation

 This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
 license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
 copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

 This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.
*/

import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { TaskComponent } from './components/task/task.component';
import { AddUpdateTaskComponent } from './components/task/add-update-task/add-update-task.component';
import { FileJobsComponent } from './components/file-jobs/file-jobs.component';
import { GrafanaComponent } from "./components/grafana/grafana.component";
import { UploadComponent } from './components/upload/upload.component';

const routes: Routes = [
  { path: '', redirectTo: 'task', pathMatch: 'full' },
  { path: 'task', component: TaskComponent },
  { path: 'task/add', component: AddUpdateTaskComponent },
  { path: 'task/update/:id', component: AddUpdateTaskComponent },
  { path: 'jobs', component: FileJobsComponent },
  { path: 'grafana', component: GrafanaComponent },
  { path: 'upload', component: UploadComponent },
  { path: '**', redirectTo: 'task' },
];

@NgModule({
  imports: [RouterModule.forRoot(routes, { scrollPositionRestoration: 'enabled' })],
  exports: [RouterModule]
})
export class AppRoutingModule { }
