/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { ComponentFixture, fakeAsync, TestBed } from '@angular/core/testing';
import { AddUpdateTaskComponent } from './add-update-task.component';
import { OpenAirPipeline } from '../../../services/data.service'
import { TaskService } from '../../../services/task.service'

import { HttpClientTestingModule } from '@angular/common/http/testing';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterTestingModule } from '@angular/router/testing';
import { MatLegacySnackBarModule as MatSnackBarModule } from "@angular/material/legacy-snack-bar";
import { MatLegacyFormFieldModule as MatFormFieldModule } from "@angular/material/legacy-form-field";
import { NoopAnimationsModule } from "@angular/platform-browser/animations";
import { MatLegacySelectModule as MatSelectModule } from "@angular/material/legacy-select";
import { MatLegacyRadioModule as MatRadioModule } from "@angular/material/legacy-radio";
import { MatLegacyProgressSpinnerModule as MatProgressSpinnerModule } from "@angular/material/legacy-progress-spinner";
import { MatLegacyListModule as MatListModule } from "@angular/material/legacy-list";
import { MatLegacyInputModule as MatInputModule } from "@angular/material/legacy-input";
import { MatIconModule } from "@angular/material/icon";
import { MatLegacyButtonModule as MatButtonModule } from "@angular/material/legacy-button";

import { Observable, of } from 'rxjs';

describe('AddUpdateTaskComponent', () => {
  let component: AddUpdateTaskComponent;
  let fixture: ComponentFixture<AddUpdateTaskComponent>;
  let taskServiceStub: Partial<TaskService>;
  let taskService: Partial<TaskService>;

  beforeEach(async () => {
    taskServiceStub = {
      getOpenAirPipelines(): Observable<OpenAirPipeline[]> { return of([]) },
    };

    await TestBed.configureTestingModule({
      declarations: [ AddUpdateTaskComponent ],
      imports: [
        HttpClientTestingModule,
        NoopAnimationsModule,
        FormsModule,
        ReactiveFormsModule,
        RouterTestingModule,
        MatFormFieldModule,
        MatRadioModule,
        MatSelectModule,
        MatListModule,
        MatInputModule,
        MatIconModule,
        MatButtonModule,
        MatSnackBarModule,
        MatProgressSpinnerModule,
      ],
      providers: [{provide: TaskService, useValue: taskServiceStub}]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(AddUpdateTaskComponent);
    component = fixture.componentInstance;
    taskService = TestBed.inject(TaskService);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize form page when creating a new Task', () => {
    const mockPipelines: OpenAirPipeline[] = [{
        "Id": "cbdf8545-8a43-4bbe-b7f2-9ad3575f6896",
        "Name": "MockName",
        "Description": "MockDescription",
        "SubscriptionTopic": "mock-topic",
        "Status": "Running"
    }]

    spyOn(taskService, 'getOpenAirPipelines').and.returnValue(of(mockPipelines));

    component.ngOnInit();
    expect(taskService.getOpenAirPipelines).toHaveBeenCalled();
    expect(component.jobSelectorTypes).toEqual(['matches', 'contains']);
    expect(component.addUpdateForm).not.toBeNull();
  });

  it('should display the Add title in a h2 tag when id is not set', fakeAsync(() => {
    component.id = undefined;
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.querySelector('h2').textContent).toContain('Add Task');
  }));

  it('should display Update title in a h2 tag when id is set', fakeAsync(() => {
    component.id = '11b91fbb-f812-483c-a154-c77f5dc4136d';
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.querySelector('h2').textContent).toContain('Update Task');
  }));

  it('should check if form is invalid when some of the required fields are empty', () => {
    component.addUpdateForm.controls.Description.setValue('');
    expect(component.addUpdateForm.valid).toBeFalsy();
  });

  it('should check if form is valid if only the required fields are set', () => {
    component.addUpdateForm.reset();
    component.addUpdateForm.controls.Description.setValue('New Task');
    component.addUpdateForm.controls.JobSelector.setValue('matches');
    component.addUpdateForm.controls.JobSelectorFile.setValue('test-file.tiff');
    component.addUpdateForm.controls.PipelineId.setValue('OnlyFile');
    component.addUpdateForm.controls.ResultFileFolder.setValue('output');
    component.addUpdateForm.controls.ModelParameters.setValue('{"Brightness": "0"}');

    expect(component.addUpdateForm.valid).toBeTruthy();
  });

});
