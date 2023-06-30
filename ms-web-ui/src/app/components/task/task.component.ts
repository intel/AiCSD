/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { Component, Inject, OnInit, ViewChild} from '@angular/core';
import { MatLegacyPaginator as MatPaginator } from '@angular/material/legacy-paginator';
import { MatSort } from '@angular/material/sort';
import { MatLegacyTableDataSource as MatTableDataSource } from '@angular/material/legacy-table';
import { SelectionModel } from '@angular/cdk/collections';

import { MAT_LEGACY_DIALOG_DATA as MAT_DIALOG_DATA, MatLegacyDialog as MatDialog } from '@angular/material/legacy-dialog';
import { MatLegacySnackBar as MatSnackBar } from "@angular/material/legacy-snack-bar";

import { TaskDetails, DataService } from '../../services/data.service';
import { TaskService } from "../../services/task.service";

@Component({
  selector: 'app-task',
  templateUrl: './task.component.html',
  styleUrls: ['./task.component.css']
})

export class TaskComponent implements OnInit {
  allTasks: TaskDetails[] | undefined;
  dataSource: MatTableDataSource<TaskDetails> | undefined;
  cancelClicked = false;
  displayCols: string[] = [
    'select',
    'Description', 'JobSelector',
    'PipelineId', 'ResultFileFolder', 'ModelParameters',
    'update'
  ];
  selection = new SelectionModel<TaskDetails>(true, []);

  @ViewChild(MatPaginator) paginator: MatPaginator;
  @ViewChild(MatSort) sort: MatSort;

  constructor(
    public data: DataService,
    public taskService: TaskService,
    public dialog: MatDialog,
  ) {
    this.data.currentPage = 'task';
  }

  ngOnInit(): void {
    this.taskService.getAll().subscribe((data: TaskDetails[]) => {

      //UX readable display of job selector on UI
      if(!data) {
        this.allTasks = [];
        this.dataSource = new MatTableDataSource(this.allTasks);

      } else {
        for (let task of data) {

          if(task.JobSelector.search("==") > -1){

            let arr = task.JobSelector.split(",");
            let val = arr[1].slice(0,arr[1].length - 3)
            task.JobSelector = "matches: "+ val;

          } else if (task.JobSelector.search("in") > -1) {

            let arr = task.JobSelector.split(",");
            let arr1 = arr[0].split("[");
            task.JobSelector = "contains: " + arr1[1];

          }
        }

        this.allTasks = data;
        this.dataSource = new MatTableDataSource(data);
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

  openDeleteDialog(tasks: TaskDetails[]): void {
    const dialogRef = this.dialog.open(DeleteTaskDialog, {
      data: tasks,
    });

    dialogRef.afterClosed().subscribe((toDelete: TaskDetails[]) => {
      this.handleDelete(toDelete);
    });
  }

  handleDelete(toDelete: TaskDetails[]): void {
    if (toDelete.length === 0) {
        return;
      }

      toDelete.forEach(deleted => this.taskService.delete(deleted.Id).subscribe(
        () => {
          this.allTasks = this.allTasks.filter(t => !toDelete.includes(t));
          this.dataSource = new MatTableDataSource(this.allTasks)
        }
      ))
      this.selection.clear();
  }

  allSelected(): boolean {
    const numSelected = this.selection.selected.length;
    const numRows = this.allTasks?.length;
    return numRows > 0 && numSelected === numRows;
  }

  toggleAll(): void {
    this.allSelected() ?
      this.selection.clear() :
      this.allTasks?.forEach(row => this.selection.select(row));
  }

  checkboxLabel(row?: TaskDetails): string {
    if (!row) {
      return `${this.allSelected() ? 'Select' : 'Deselect'} all`;
    }
    return `${this.selection.isSelected(row) ? 'Deselect' : 'Select'} ${row.Description}`;
  }

}

@Component({
  selector: 'delete-task-dialog',
  templateUrl: 'delete-task-dialog.html',
})
export class DeleteTaskDialog {
  constructor(@Inject(MAT_DIALOG_DATA) public data: TaskDetails[]) {
  }
}
