/* Copyright 2023 Intel Corporation

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { Model } from '../../services/data.service';
import { ModelService } from "../../services/model.service";

@Component({
  selector: 'app-upload',
  templateUrl: './upload.component.html',
  styleUrls: ['./upload.component.css']
})
export class UploadComponent implements OnInit {
  modelForm: FormGroup
  submitted = false
  zipFile: File

  constructor(
    public fb: FormBuilder,
    private router: Router,
    public modelService: ModelService,
    private cd: ChangeDetectorRef,
  ) { }

  ngOnInit(): void {
    this.modelForm = this.fb.group({
      ModelName: ['', [Validators.required, Validators.maxLength(80)]],
      ModelType: ['ovms'],
      ModelZip: [null, [Validators.required, Validators.pattern(/\.zip$/)]]
    })
  }

  onFileChange(event) {
    const reader = new FileReader();
 
    if(event.target.files && event.target.files.length) {
      const [file] = event.target.files;
      reader.readAsDataURL(file);
  
      reader.onload = () => {
        this.modelForm.patchValue({
          ModelZip: reader.result
       });
      
        // need to run CD since file load runs outside of zone
        this.cd.markForCheck();
      };
    }
  }

  onSubmit() {

    if (!this.modelForm.valid) {
      return;
    }

    this.submitted = true

    let newModel : Model = {
        Name: this.modelForm.get('ModelName').value,
        Type: this.modelForm.get('ModelType').value,
        Zip: this.modelForm.get("ModelZip").value,
    }

    this.modelService.add(newModel).subscribe()
    this.router.navigateByUrl('/task')
    this.submitted = false
  }
}
