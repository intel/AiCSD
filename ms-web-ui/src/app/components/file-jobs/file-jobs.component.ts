/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

import { Component, OnInit, ViewChild } from '@angular/core';
import { animate, state, style, transition, trigger } from '@angular/animations';

import { MatLegacyTableDataSource as MatTableDataSource } from '@angular/material/legacy-table';
import { MatLegacyPaginator as MatPaginator } from '@angular/material/legacy-paginator';
import { MatSort } from '@angular/material/sort';
import { MatLegacyDialog as MatDialog } from '@angular/material/legacy-dialog';

import { interval } from 'rxjs';

import { Job, DataService, Verification } from './../../services/data.service';
import { FileJobsService } from "./../../services/file-jobs.service";
import { JobFilesDialogComponent } from './job-files-dialog.component';
import { JobImagesComponent } from "./job-images.component";

@Component({
  selector: 'app-file-jobs',
  templateUrl: './file-jobs.component.html',
  styleUrls: ['./file-jobs.component.css'],
  animations: [
    trigger('detailExpand', [
      state('collapsed', style({ height: '0px', minHeight: '0' })),
      state('expanded', style({ height: '*' })),
      transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
  ],
})

export class FileJobsComponent implements OnInit {
  allJobs: Job[] | undefined;
  dataSource: MatTableDataSource<Job> | undefined;
  displayCols: string[] = ['JobDetails', 'Owner', 'Status', 'InputFileName',  'OutputFileHost', 'Verification', 'PipelineDetailsStatus', 'PipelineDetailsQCFlags', 'Results', 'ErrorDetails', 'LastUpdated'];
  expandedFiles: boolean | false;

  @ViewChild(MatPaginator) paginator: MatPaginator;
  @ViewChild(MatSort) sort: MatSort;

  constructor(
    public data: DataService,
    public fileJobsService: FileJobsService,
    public dialog: MatDialog
  ) {
    this.data.currentPage = 'jobs';
  }

  ngOnInit(): void {
    this.loadPage()
    const reloadTime = 10000 // 10000 ms = 10 seconds
    interval(reloadTime).subscribe(() => { this.updateData() })
  }

  updateData() {
    this.fileJobsService.getAll().subscribe((data: Job[]) => {
      let newJobs: Job[] = []
      for (const job of data) {
        if (!this.dataSource.data.some(j => j.Id === job.Id)) {
          newJobs.push(job)
        }
      }

      this.dataSource.data = this.dataSource.data.concat(newJobs)
    })
  }

  loadPage(): void {
    this.fileJobsService.getAll().subscribe((data: Job[]) => {
      if (!data) {
        this.allJobs = [];
        this.dataSource = new MatTableDataSource(this.allJobs);
      } else {
        this.allJobs = data;
        this.dataSource = new MatTableDataSource(data);
      }

      this.dataSource.filterPredicate = function (data: Job, filter: string) {
        const pendingSearchTerm: string = "pending"
        const acceptSearchTerm: string = "accepted"
        const rejectSearchTerm: string = "rejected"
        filter = filter.trim().toLowerCase()

        let searchSpace: string[] = [
          data.Owner,
          data.Status,
          data.PipelineDetails.Status,
          data.InputFile.Name
        ]

        for (var searchTerm of searchSpace) {
          if (searchTerm.trim().toLowerCase().includes(filter)) {
            return true
          }
        }

        if (pendingSearchTerm.includes(filter) && data.Verification === Verification.Pending) {
          return true
        } else if (acceptSearchTerm.includes(filter) && data.Verification === Verification.Accepted) {
          return true
        } else if (rejectSearchTerm.includes(filter) && data.Verification === Verification.Rejected) {
          return true
        }

        return false
      }

      this.dataSource.paginator = this.paginator;
      this.dataSource.sort = this.sort;

      //handle sorting for nested datatypes - PipelineDetails & InputFile
      this.dataSource.sortingDataAccessor = (item, property) => {
        switch (property) {
          case 'PipelineDetailsQCFlags': return item.PipelineDetails.QCFlags;
          case 'PipelineDetailsStatus': return item.PipelineDetails.Status;
          case 'InputFileName': return item.InputFile.Name;
          default: return item[property];
        }
      }
    })
  }

  toggleTableRows() {
    this.expandedFiles = !this.expandedFiles;

    this.dataSource.data.forEach((row: any) => {
      row.expandedFiles = this.expandedFiles;
    })
  }

  applyFilter(event: Event) {
    const filterValue = (event.target as HTMLInputElement).value;
    this.dataSource.filter = filterValue.trim().toLowerCase();

    if (this.dataSource.paginator) {
      this.dataSource.paginator.firstPage();
    }
  }

  openFilesDialog(row: Job) {
    const dialogRef = this.dialog.open(JobFilesDialogComponent, {
      // Can be closed only by clicking the close button
      disableClose: true,
      data: row
    });
  }

  openImagesDialog(row: Job) {
    const dialogRef = this.dialog.open(JobImagesComponent, {
      // Can be closed only by clicking one of the verification buttons (pressing ESC defaults to pending)
      disableClose: false,
      data: row
    });

    let verificationBefore = row.Verification

    dialogRef.afterClosed().subscribe(result => {
      switch (result) {
        case "accept":
          row.Verification = Verification.Accepted
          break
        case "reject":
          row.Verification = Verification.Rejected
          break
        default:
          row.Verification = Verification.Pending;
      }

      if (row.Verification != verificationBefore) {
        this.fileJobsService.updateVerification(row.Id, row.Verification).subscribe()
        if (row.Verification == Verification.Rejected) {
          this.fileJobsService.addToRejectedDir(row.Id).subscribe()
        } else if (verificationBefore == Verification.Rejected) {
          this.fileJobsService.removeFromRejectedDir(row.Id).subscribe()
        }
      }
    });
  }

  getStatusColor(row: Job) {
    switch (row.Status) {
      case 'Complete':
        return 'green'
      case 'PipelineError':
      case 'NoPipelineFound':
      case 'TransmissionFailed':
      case 'FileErrored':
        return 'red'
      default:
        return 'primary'
    }
  }

  getVerificationIcon(row: Job) {
    switch (row.Verification) {
      case Verification.Pending:
        return 'circle'
      case Verification.Accepted:
        return 'check_circle'
      case Verification.Rejected:
        return 'cancel'
    }
  }

  getVerificationColor(row: Job) {
    switch (row.Verification) {
      case Verification.Pending:
        return 'gray'
      case Verification.Accepted:
        return 'green'
      case Verification.Rejected:
        return 'red'
    }
  }
}
