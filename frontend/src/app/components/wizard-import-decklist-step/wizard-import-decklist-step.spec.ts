import { ComponentFixture, TestBed } from '@angular/core/testing';

import { WizardImportDecklistStep } from './wizard-import-decklist-step';

describe('WizardImportDecklistStep', () => {
  let component: WizardImportDecklistStep;
  let fixture: ComponentFixture<WizardImportDecklistStep>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [WizardImportDecklistStep]
    })
    .compileComponents();

    fixture = TestBed.createComponent(WizardImportDecklistStep);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
