/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

import { Component, OnInit, Inject} from '@angular/core';
import {Job, InputFile, OutputFile} from "../../services/data.service";
import {FileJobsService} from "../../services/file-jobs.service";
import {MAT_LEGACY_DIALOG_DATA as MAT_DIALOG_DATA} from "@angular/material/legacy-dialog";

@Component({
  selector: 'app-job-images',
  templateUrl: './job-images.component.html',
  styleUrls: ['./job-images.component.css']
})
export class JobImagesComponent implements OnInit {

  selectedJob: Job | undefined;
  inputFile: InputFile | undefined;
  outputFiles: OutputFile[] | undefined;

  constructor(
    public fileJobsService: FileJobsService,
    @Inject(MAT_DIALOG_DATA) public data: any,
    ) { }

  ngOnInit(): void {
    this.fileJobsService.get(this.data.Id).subscribe( (job: Job) => {
      if(!job) {
        this.outputFiles = [];
      } else {
        this.selectedJob = job;
        this.inputFile = job.InputFile;
        this.outputFiles = job.PipelineDetails.OutputFiles;
      }
    })
  }

}
