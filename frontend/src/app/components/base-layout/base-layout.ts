import { Component } from '@angular/core';
import {MatToolbar} from '@angular/material/toolbar';
import {RouterOutlet} from '@angular/router';

@Component({
  selector: 'app-base-layout',
  imports: [
    MatToolbar,
    RouterOutlet
  ],
  templateUrl: './base-layout.html',
  styleUrl: './base-layout.scss'
})
export class BaseLayout {

}
