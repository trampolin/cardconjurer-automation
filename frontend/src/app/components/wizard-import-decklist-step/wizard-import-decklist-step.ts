import {Component, inject, Input} from '@angular/core';
import {FormBuilder, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {MatFormField, MatInput} from '@angular/material/input';
import {MatError} from '@angular/material/form-field';

@Component({
  selector: 'app-wizard-import-decklist-step',
  imports: [
    MatInput,
    MatFormField,
    MatError,
    ReactiveFormsModule
  ],
  templateUrl: './wizard-import-decklist-step.html',
  styleUrl: './wizard-import-decklist-step.scss'
})
export class WizardImportDecklistStep {
  fb = inject(FormBuilder);

  formGroup: FormGroup = this.fb.group({
    deckLink: ['',

    ]
  });
}
