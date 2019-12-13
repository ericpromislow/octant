// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import {
  Component,
  Input,
  OnChanges,
  OnInit,
  SimpleChanges,
} from '@angular/core';
import { ListView, View } from 'src/app/models/content';

import { IconService } from '../../services/icon.service';
import { ViewService } from '../../services/view/view.service';
import { DynamicView } from '../dynamic-view/dynamic-view';

@Component({
  selector: 'app-view-list',
  templateUrl: './list.component.html',
  styleUrls: ['./list.component.scss'],
})
export class ListComponent extends DynamicView implements OnInit, OnChanges {
  @Input() listView: ListView;
  title: string;

  iconName: string;

  constructor(
    private iconService: IconService,
    private viewService: ViewService
  ) {
    super();
  }

  identifyItem = (index: number, item: View): string => {
    return this.viewService.titleAsText(item.metadata.title);
  };

  ngOnInit(): void {}

  ngOnChanges(changes: SimpleChanges): void {
    if (changes.listView) {
      console.log(`list view updated`, changes.listView);
      const { currentValue } = changes.listView;

      this.title = this.viewService.viewTitleAsText(currentValue);
      this.iconName = this.iconService.load(currentValue.config);
    }
  }
}
