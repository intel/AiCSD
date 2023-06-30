/* INTEL CONFIDENTIAL

 Copyright (C) 2023 Intel Corporation

 This software and the related documents are Intel copyrighted materials, and your use of them is governed by the express
 license under which they were provided to you ("License"). Unless the License provides otherwise, you may not use, modify,
 copy, publish, distribute, disclose or transmit this software or the related documents without Intel's prior written permission.

 This software and the related documents are provided as is, with no express or implied warranties, other than those that are expressly stated in the License.
*/

import { Component } from '@angular/core';
import { BreakpointObserver } from "@angular/cdk/layout";
import { DataService } from "./services/data.service";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  public title = 'AiCSD Sample UI';
  public navbarCollapseShow = false;
  darkTheme: boolean;

  constructor(public data: DataService, private bo: BreakpointObserver) {
    bo.observe('(prefers-color-scheme: dark)').subscribe((state) => {
      this.useDarkTheme = state.matches;
    });
  }

  get useDarkTheme() {
    return this.darkTheme;
  }

  set useDarkTheme(v: boolean) {
    this.darkTheme = v;
    if (v) {
      document.body.classList.add('dark-theme');
    } else {
      document.body.classList.remove('dark-theme');
  }
  }
}
