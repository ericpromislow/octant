/*
 * Copyright (c) 2019 VMware, Inc. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import { Component, Input, OnInit } from '@angular/core';
import { DonutChartView, DonutSegment } from '../../../../models/content';

export interface SegmentDescriptor {
  array: string;
  className: string;
  offset: number;
}

@Component({
  selector: 'app-view-donut-chart',
  templateUrl: './donut-chart.component.html',
  styleUrls: ['./donut-chart.component.scss'],
})
export class DonutChartComponent implements OnInit {
  @Input() view: DonutChartView;

  @Input() circumference = 100;
  @Input() height = 42;

  constructor() {}

  ngOnInit() {}

  trackByDescriptor(index: number, item: SegmentDescriptor) {
    if (!item) {
      return null;
    }

    return item.className;
  }

  radius(): number {
    return this.circumference / (2 * Math.PI);
  }

  viewBox(): string {
    return `0 0 ${this.height} ${this.height}`;
  }

  center(): number {
    return this.height / 2;
  }

  itemCount(): number {
    if (!this.view) {
      return 0;
    }
    return this.view.config.segments.reduce<number>(
      (a: number, s: DonutSegment) => a + s.count,
      0
    );
  }

  itemLabel(): string {
    if (!this.view) {
      return '';
    }

    return this.itemCount() > 1
      ? this.view.config.labels.plural
      : this.view.config.labels.singular;
  }

  descriptors(): SegmentDescriptor[] {
    let offset = 0;

    if (!this.view) {
      return [];
    }

    return this.view.config.segments
      .sort((a, b) => (a.status > b.status ? 1 : -1))
      .map<SegmentDescriptor>(s => {
        const x = (s.count / this.itemCount()) * 100;

        const curOffset = offset;
        offset += 100 - x;
        return {
          array: `${x} ${100 - x}`,
          offset: curOffset,
          className: `donut-segment ${s.status}`,
        };
      });
  }
}
