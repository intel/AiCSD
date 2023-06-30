/* INTEL CONFIDENTIAL

 Copyright (C) 2023 Intel Corporation

 This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
 license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
 copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

 This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.
*/

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
