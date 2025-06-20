import { Component } from '@angular/core';
import {BaseLayout} from './components/base-layout/base-layout';

@Component({
  selector: 'app-root',
  imports: [BaseLayout],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App {
  protected title = 'frontend';
}
