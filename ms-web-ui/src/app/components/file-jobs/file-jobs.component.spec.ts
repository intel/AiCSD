/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FileJobsComponent } from './file-jobs.component';

import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { FileJobsService } from './../../services/file-jobs.service';

import { Observable, of } from 'rxjs';
import { Job, Verification } from './../../services/data.service';
import { NoopAnimationsModule } from "@angular/platform-browser/animations";
import { MatLegacyCheckboxModule as MatCheckboxModule } from "@angular/material/legacy-checkbox";
import { MatLegacyDialogModule as MatDialogModule } from "@angular/material/legacy-dialog";
import { MatLegacyInputModule as MatInputModule } from '@angular/material/legacy-input';
import { MatLegacyFormFieldModule as MatFormFieldModule } from "@angular/material/legacy-form-field";
import { MatLegacySnackBarModule as MatSnackBarModule } from "@angular/material/legacy-snack-bar";
import { MatLegacyTableModule as MatTableModule } from "@angular/material/legacy-table";
import { MatLegacyPaginatorModule as MatPaginatorModule } from '@angular/material/legacy-paginator';
import { MatSortModule } from '@angular/material/sort';

describe('FileJobsComponent', () => {
  let component: FileJobsComponent;
  let fixture: ComponentFixture<FileJobsComponent>;
  let fileJobsServiceStub: Partial<FileJobsService>;
  let fileJobsService: FileJobsService;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    fileJobsServiceStub = {
      getAll(): Observable<Job[]> { return of([]) },
    };

    await TestBed.configureTestingModule({
      declarations: [ FileJobsComponent ],
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
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(FileJobsComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    fileJobsService = TestBed.inject(FileJobsService);
    fixture.detectChanges();

  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should get jobs using the service and set it to component property', () => {
    const configs: Job[] = [{
        Id: '123asd',
        Owner: 'none',
        Status: 'Complete',
        InputFile: {
            Hostname: 'testHostname1',
            DirName: '/tmp/test1',
            Name: 'test-image1',
            ArchiveName: '/tmp/archive/test-image1_archive.tiff',
            Viewable: '/tmp/archive/test-image1_archive.jpeg',
            Extension: 'tiff',
            Attributes: {
                'LabName': 'Microscope',
                'Labequipment': 'ScienceLab',
                'Operator': 'Scientist 1'
            }
        },
        PipelineDetails: {
            TaskId: '123',
            Status: 'Complete',
            QCFlags: '',
            OutputFileHost: 'test',
            OutputFiles: [{
              DirName: '/tmp/result',
              Name: '/tmp/result/test1.tiff',
              ArchiveName: '/tmp/archive/test1out_archive.tiff',
              Viewable: '/tmp/archive/test1out_archive.jpeg',
              Extension: '.tiff',
              Status: 'FileComplete',
              ErrorDetails: {
                Owner: 'file-receiver-oem',
                Error: 'testError'
              },
              Owner: 'file-receiver-oem'
            }],
            Results: '',
        },
        ErrorDetails: {
          Owner: 'file-receiver-oem',
          Error:'testError'
        },
        Verification: Verification.Pending
      } as Job,
    ];

    spyOn(fileJobsService, 'getAll').and.returnValue(of(configs));

    component.ngOnInit();
    expect(fileJobsService.getAll).toHaveBeenCalled();
    expect(component.allJobs).toEqual(configs);
  });

  it('should get jobs using the service and set it to component property for multiple files', () => {
    const configs: Job[] = [{
        Id: '123asd',
        Owner: 'none',
        Status: 'Complete',
        InputFile: {
            Hostname: 'testHostname1',
            DirName: '/tmp/test1',
            Name: 'test-image1',
            ArchiveName: '/tmp/archive/test-image1_archive.tiff',
            Viewable: '/tmp/archive/test-image1_archive.jpeg',
            Extension: 'tiff',
            Attributes: {
                'LabName': 'Microscope',
                'Labequipment': 'ScienceLab',
                'Operator': 'Scientist 1'
            }
        },
        PipelineDetails: {
            TaskId: '123',
            Status: 'Complete',
            QCFlags: '',
            OutputFileHost: 'test',
            OutputFiles: [{
              DirName: '/tmp/result',
              Name: '/tmp/result/test1.tiff',
              ArchiveName: '/tmp/archive/test1_archive.tiff',
              Viewable: '/tmp/archive/test1_archive.jpeg',
              Extension: '.tiff',
              Status: 'FileComplete',
              ErrorDetails: {
                Owner: 'file-receiver-oem',
                Error:'testError'
              },
              Owner: 'file-receiver-oem'
            }, {
              DirName: '/tmp/result',
              Name: '/tmp/result/test2.tiff',
              Extension: '.tiff',
              Status: 'FileTransmissionFailed',
              ErrorDetails: {
                Owner: 'file-receiver-oem',
                Error:'failed to transmit file...'
              },
              Owner: 'file-receiver-oem'
            }],
            Results: '',
        },
      ErrorDetails: {
        Owner: 'file-receiver-oem',
        Error:'testError'
      },
      Verification: Verification.Pending
      } as Job,
    ];

    spyOn(fileJobsService, 'getAll').and.returnValue(of(configs));

    component.ngOnInit();
    expect(fileJobsService.getAll).toHaveBeenCalled();
    expect(component.allJobs).toEqual(configs);
  });

});
