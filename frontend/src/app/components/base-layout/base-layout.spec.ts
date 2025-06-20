import { ComponentFixture, TestBed } from '@angular/core/testing';

import { BaseLayout } from './base-layout';

describe('BaseLayout', () => {
  let component: BaseLayout;
  let fixture: ComponentFixture<BaseLayout>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BaseLayout]
    })
    .compileComponents();

    fixture = TestBed.createComponent(BaseLayout);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
