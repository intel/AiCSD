/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

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
