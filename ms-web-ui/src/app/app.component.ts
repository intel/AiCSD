/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

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
