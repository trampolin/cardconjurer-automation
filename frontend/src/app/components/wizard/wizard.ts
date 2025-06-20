import { Component } from '@angular/core';
import {MatStep, MatStepLabel, MatStepper, MatStepperNext, MatStepperPrevious} from '@angular/material/stepper';
import {MatButton} from '@angular/material/button';
import {ReactiveFormsModule, FormBuilder, FormGroup, Validators} from '@angular/forms';
import {MatInputModule} from '@angular/material/input';
import {WizardImportDecklistStep} from '../wizard-import-decklist-step/wizard-import-decklist-step';

@Component({
  selector: 'app-wizard',
  imports: [
    MatStepper,
    MatStep,
    MatButton,
    ReactiveFormsModule,
    MatInputModule,
    MatStepLabel,
    MatStepperNext,
    MatStepperPrevious,
    WizardImportDecklistStep,
  ],
  templateUrl: './wizard.html',
  styleUrl: './wizard.scss'
})
export class Wizard  {

}
