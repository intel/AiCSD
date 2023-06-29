/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TaskComponent } from './task.component';
import { TaskDetails } from '../../services/data.service'
import { TaskService } from '../../services/task.service'

import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';

import { NoopAnimationsModule } from "@angular/platform-browser/animations";
import { MatLegacyCheckboxModule as MatCheckboxModule } from "@angular/material/legacy-checkbox";
import { MatLegacyDialogModule as MatDialogModule } from "@angular/material/legacy-dialog";
import { MatLegacyDialogRef as MatDialogRef } from '@angular/material/legacy-dialog';
import { MatLegacyInputModule as MatInputModule } from '@angular/material/legacy-input';
import { MatLegacyFormFieldModule as MatFormFieldModule } from "@angular/material/legacy-form-field";
import { MatLegacySnackBarModule as MatSnackBarModule } from "@angular/material/legacy-snack-bar";
import { MatLegacyTableModule as MatTableModule } from "@angular/material/legacy-table";
import { MatLegacyPaginatorModule as MatPaginatorModule } from '@angular/material/legacy-paginator';
import { MatSortModule } from '@angular/material/sort';

import { Observable, of } from 'rxjs';

describe('TaskComponent', () => {
  let component: TaskComponent;
  let fixture: ComponentFixture<TaskComponent>;
  let httpMock: HttpTestingController;
  let taskServiceStub: Partial<TaskService>;
  let taskService: Partial<TaskService>;

  beforeEach(async () => {
    taskServiceStub = {
      getAll(): Observable<TaskDetails[]> { return of([]) },
      delete(id): Observable<any> { return of(null); },
    };

    await TestBed.configureTestingModule({
      declarations: [ TaskComponent ],
      imports: [
        HttpClientTestingModule,
        NoopAnimationsModule,
        MatCheckboxModule,
        MatDialogModule,
        MatInputModule,
        MatFormFieldModule,
        MatSnackBarModule,
        MatPaginatorModule,
        MatSortModule,
        MatTableModule,
      ],
      providers: [{provide: TaskService, useValue: taskServiceStub}]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(TaskComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    taskService = TestBed.inject(TaskService);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should get all tasks using the task service and set it to component property', () => {
    const mockTasks: TaskDetails[] = [{
        Id: '11b91fbb-f812-483c-a154-c77f5dc4136d',
        Description: 'Generate Output File',
        JobSelector: 'matches: test-image1.tiff',
        PipelineId: 'only-file',
        ResultFileFolder: '/tmp/files/output',
        ModelParameters: JSON.parse('{ "Brightness": "0", "Resolution": "256" }'),
        LastUpdated: 0
      }, {
        Id: 'd09abd02-2b7a-4f7a-8454-6270f3569147',
        Description: 'Generate Result',
        JobSelector: 'contains: test-image2.tiff',
        PipelineId: 'only-results',
        ResultFileFolder: '/tmp/files/output',
        ModelParameters: JSON.parse('{ "Brightness": "0", "Resolution": "128" }'),
        LastUpdated: 0
      }
    ];

    spyOn(taskService, 'getAll').and.returnValue(of(mockTasks));

    component.ngOnInit();
    expect(taskService.getAll).toHaveBeenCalled();
    expect(component.allTasks).toEqual(mockTasks);
  });

  it('should delete a task when delete is called', () => {
    const mockTasks: TaskDetails[] = [{
        Id: 'd09abd02-2b7a-4f7a-8454-6270f3569147',
        Description: 'Generate Result',
        JobSelector: 'contains: test-image2.tiff',
        PipelineId: 'only-results',
        ResultFileFolder: '/tmp/files/output',
        ModelParameters: JSON.parse('{ "Brightness": "0", "Resolution": "128" }'),
        LastUpdated: 0
      }
    ];

    component.allTasks = mockTasks;

    spyOn(component.dialog, 'open')
      .and
      .returnValue({
        afterClosed: () => of(true)
      } as MatDialogRef<typeof component>);

    spyOn(taskService, 'delete').and.callThrough();

    component.handleDelete([{
      Id: 'd09abd02-2b7a-4f7a-8454-6270f3569147',
      Description: 'Generate Result',
      JobSelector: 'contains: test-image2.tiff',
      PipelineId: 'only-results',
      ResultFileFolder: '/tmp/files/output',
      ModelParameters: JSON.parse('{ "Brightness": "0", "Resolution": "128" }'),
      LastUpdated: 0
    }])

    expect(taskService.delete).toHaveBeenCalledWith('d09abd02-2b7a-4f7a-8454-6270f3569147');
    expect(component.dialog).toBeDefined();
    expect(component.selection.hasValue()).toBeFalse();
  });
});
