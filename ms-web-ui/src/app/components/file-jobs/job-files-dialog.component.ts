/* INTEL CONFIDENTIAL

 Copyright (C) 2023 Intel Corporation

 This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
 license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
 copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

 This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.
*/

import { Component, OnInit, Inject, ViewChild } from '@angular/core';

import { MatLegacyTableDataSource as MatTableDataSource } from '@angular/material/legacy-table';
import { MatLegacyPaginator as MatPaginator } from '@angular/material/legacy-paginator';
import { MatSort } from '@angular/material/sort';
import { MatLegacyDialogRef as MatDialogRef, MAT_LEGACY_DIALOG_DATA as MAT_DIALOG_DATA } from '@angular/material/legacy-dialog';

import { Job, OutputFile } from '../../services/data.service';
import { FileJobsService } from "../../services/file-jobs.service";

@Component({
  selector: 'app-job-files-dialog',
  templateUrl: './job-files-dialog.component.html',
  styleUrls: ['./job-files-dialog.component.css'],
})

export class JobFilesDialogComponent implements OnInit {
  selectedJob: Job | undefined;
  allFiles: OutputFile[] | undefined;
  dataSource: MatTableDataSource<OutputFile> | undefined;
  displayCols: string[] = ['status', 'path'];

  @ViewChild(MatPaginator) paginator: MatPaginator;
  @ViewChild(MatSort) sort: MatSort;

  constructor(
    public fileJobsService: FileJobsService,
    public dialogRef: MatDialogRef<JobFilesDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: any,
  ) {
  }

  ngOnInit(): void {
    this.fileJobsService.get(this.data.Id).subscribe((job: Job) => {
      if(!job) {
        this.allFiles = [];
        this.dataSource = new MatTableDataSource(this.allFiles);
      } else {
        this.selectedJob = job;
        this.allFiles = job.PipelineDetails.OutputFiles;
        this.dataSource = new MatTableDataSource(this.allFiles);
      }

      this.dataSource.paginator = this.paginator;
      this.dataSource.sort = this.sort;
    });
  }

  applyFilter(event: Event) {
    const filterValue = (event.target as HTMLInputElement).value;
    this.dataSource.filter = filterValue.trim().toLowerCase();

    if (this.dataSource.paginator) {
      this.dataSource.paginator.firstPage();
    }
  }
}
