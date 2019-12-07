import { Component, Input, OnInit } from '@angular/core';
import { SingleStatView } from '../../../../models/content';

@Component({
  selector: 'app-single-stat',
  templateUrl: './single-stat.component.html',
  styleUrls: ['./single-stat.component.scss'],
})
export class SingleStatComponent implements OnInit {
  @Input() view: SingleStatView;

  constructor() {}

  ngOnInit() {}
}
