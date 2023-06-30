/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { Component, OnInit } from '@angular/core';
import { Validators, UntypedFormBuilder, UntypedFormGroup } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { MatLegacySnackBar as MatSnackBar } from "@angular/material/legacy-snack-bar";

import { TaskDetails, OpenAirPipeline } from '../../../services/data.service';
import { TaskService } from "../../../services/task.service";
import { catchError } from 'rxjs/operators';

@Component({
  selector: 'app-add-update-task',
  templateUrl: './add-update-task.component.html',
  styleUrls: ['./add-update-task.component.css']
})
export class AddUpdateTaskComponent implements OnInit {
  addUpdateForm: UntypedFormGroup;
  submitted = false;
  id: string | undefined;
  availablePipelines: OpenAirPipeline[];
  jobSelectorTypes: string[];

  constructor(
    public fb: UntypedFormBuilder,
    private router: Router,
    private route: ActivatedRoute,
    private snackBar: MatSnackBar,
    public taskService: TaskService
  ) { }

  get isAdd(): boolean {
    return !this.id;
  }

  ngOnInit(): void {
    this.id = this.route.snapshot.params.id;

    this.addUpdateForm = this.fb.group({
      Description: ['', [Validators.required, Validators.maxLength(80)]],
      JobSelector: ['matches', [Validators.required]],
      JobSelectorFile: ['', [Validators.required, Validators.maxLength(80)]],
      PipelineId: [undefined, [Validators.required]],
      ResultFileFolder: [undefined, [Validators.max(200)]],
      ModelParameters: ['{"Brightness": "0"}', Validators.maxLength(500)],
    });

    this.taskService.getOpenAirPipelines().subscribe({
      next: (pipelines: OpenAirPipeline[]) => {
        this.availablePipelines = pipelines
      },
      error: this.errSnack('Unable to fetch available pipelines')
    });

    this.jobSelectorTypes = ['matches', 'contains']
    this.addUpdateForm.patchValue({ResultFileFolder: 'output'});

    if (!this.isAdd) {
      this.taskService.getAll().subscribe({
        next: (tasks?: TaskDetails[]) => {
          if (!tasks) {
            this.router.navigateByUrl('/task/add');
            throw 'Missing task detail data';
          }

          const task = tasks.find(task => task.Id === this.id);
          if (!task) {
            this.router.navigateByUrl('/task/add');
            throw 'Invalid task ID';
          }

          this.addUpdateForm.patchValue(task);
          this.addUpdateForm.patchValue(this.getJobSelectorFromTask(task.JobSelector));
          this.addUpdateForm.patchValue({ResultFileFolder: task.ResultFileFolder.replace('/tmp/files/', '')});
          this.addUpdateForm.patchValue({ModelParameters: JSON.stringify(task.ModelParameters)});
        },
        error: this.errSnack('Unable to fetch task details')
      });
    }
  }

  getJobSelectorFromTask(taskJobSelector: string): any {
    let js;
    let arr = taskJobSelector.split(",");
    if(taskJobSelector.search("==") > -1){
      js = "matches"
      let val = arr[1].slice(0,arr[1].length - 3)
      let trimQuotes = (val.slice(0,-2)).slice(2);

      return { JobSelector: js, JobSelectorFile: trimQuotes};

    }
    else if(taskJobSelector.search("in") > -1) {
      js = "contains"
      let val = arr[0].split("[");
      let trimQuotes = (val[1].slice(0,-1)).slice(2);

      return { JobSelector: js, JobSelectorFile: trimQuotes};
    }
    else {
      return { JobSelector: '', JobSelectorFile: ''};
    }
  }

  onSubmit(): void {
    if (!this.addUpdateForm.valid) {
      return;
    }

    this.submitted = true;

    let setTask;
    if (this.isAdd){
      setTask = this.taskService.add(this.addUpdateForm.value)
      .pipe(catchError(err => {
        if (typeof err === 'string') {
          throw 'Failed to create task: ' + err;
        } else if (err?.statusText !== '') {
          throw 'Failed to create task: ' + err.statusText;
        } else {
          throw 'Failed to create task for unknown reasons';
        }
      }));
    } else {

      setTask = this.taskService.update(this.addUpdateForm.value, this.id)
                .pipe(catchError(err => {
                  if (typeof err === 'string') {
                    throw 'Failed to task: ' + err;
                  } else if (err?.statusText !== '') {
                    throw 'Failed to update task: ' + err.statusText;
                  } else {
                    throw 'Failed to update task for unknown reasons';
                  }
                }));

    }

    setTask.subscribe(
      () => {this.router.navigateByUrl('/task'); },
      this.errSnack('Add or Update failed')
    );
    this.submitted = false;

  }

  // errSnack returns an error handler that opens a SnackBar with the error message.
  errSnack(msgPrefix: string) {
    const snackBar = this.snackBar;
    return function (err) {
      const msg = msgPrefix + ': ' + (err ? String(err) : 'unknown error');
      console.log(msg);
      snackBar.open(msg, 'Dismiss');
    }
  }

}
