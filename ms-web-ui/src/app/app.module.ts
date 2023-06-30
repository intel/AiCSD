/* INTEL CONFIDENTIAL

 Copyright (C) 2023 Intel Corporation

 This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
 license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
 copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

 This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.
 */

import { NgModule } from '@angular/core';

import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { HttpClientModule } from '@angular/common/http';

import { DragDropModule } from "@angular/cdk/drag-drop";

import { MatLegacyButtonModule as MatButtonModule } from '@angular/material/legacy-button';
import { MatLegacyCardModule as MatCardModule } from '@angular/material/legacy-card';
import { MatLegacyCheckboxModule as MatCheckboxModule } from '@angular/material/legacy-checkbox';
import { MatLegacyChipsModule as MatChipsModule } from '@angular/material/legacy-chips';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatLegacyDialogModule as MatDialogModule } from '@angular/material/legacy-dialog';
import { MatLegacyFormFieldModule as MatFormFieldModule } from '@angular/material/legacy-form-field';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatLegacyInputModule as MatInputModule } from '@angular/material/legacy-input';
import { MatLegacyListModule as MatListModule } from '@angular/material/legacy-list';
import { MatNativeDateModule } from '@angular/material/core';
import { MatLegacyPaginatorModule as MatPaginatorModule } from '@angular/material/legacy-paginator';
import { MatLegacyProgressSpinnerModule as MatProgressSpinnerModule } from '@angular/material/legacy-progress-spinner';
import { MatLegacyRadioModule as MatRadioModule } from '@angular/material/legacy-radio';
import { MatLegacySelectModule as MatSelectModule } from '@angular/material/legacy-select';
import { MatLegacySliderModule as MatSliderModule } from '@angular/material/legacy-slider';
import { MatSortModule } from "@angular/material/sort";
import { MatLegacyTableModule as MatTableModule } from '@angular/material/legacy-table';
import { MatLegacyTabsModule as MatTabsModule } from '@angular/material/legacy-tabs';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatLegacySnackBarModule as MatSnackBarModule } from '@angular/material/legacy-snack-bar';

import { AppRoutingModule } from './app-routing.module';

import { AppComponent } from './app.component';
import { TaskComponent } from './components/task/task.component';
import { DeleteTaskDialog } from './components/task/task.component';
import { AddUpdateTaskComponent } from './components/task/add-update-task/add-update-task.component';
import { FileJobsComponent } from './components/file-jobs/file-jobs.component';
import { JobFilesDialogComponent } from './components/file-jobs/job-files-dialog.component';
import { GrafanaComponent } from './components/grafana/grafana.component';
import { UploadComponent } from './components/upload/upload.component';
import { JobImagesComponent } from './components/file-jobs/job-images.component';

@NgModule({
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    FormsModule, ReactiveFormsModule,
    HttpClientModule,
    DragDropModule,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatChipsModule,
    MatDatepickerModule,
    MatDialogModule,
    MatFormFieldModule,
    MatGridListModule,
    MatIconModule,
    MatInputModule,
    MatListModule,
    MatNativeDateModule,
    MatPaginatorModule,
    MatProgressSpinnerModule,
    MatRadioModule,
    MatSelectModule,
    MatSliderModule,
    MatSortModule,
    MatTableModule,
    MatTabsModule,
    MatToolbarModule,
    MatSnackBarModule,
    AppRoutingModule,
  ],
  declarations: [
    AppComponent,
    TaskComponent,
    DeleteTaskDialog,
    AddUpdateTaskComponent,
    FileJobsComponent,
    JobFilesDialogComponent,
    GrafanaComponent,
    UploadComponent,
    JobImagesComponent,
  ],
  providers: [],
  bootstrap: [AppComponent]
})

export class AppModule {}
