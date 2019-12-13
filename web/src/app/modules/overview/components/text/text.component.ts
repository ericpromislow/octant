/*
 * Copyright (c) 2019 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  ChangeDetectionStrategy,
  Component,
  Input,
  OnInit,
} from '@angular/core';
import { TextView } from 'src/app/models/content';
import { DynamicView } from '../dynamic-view/dynamic-view';

@Component({
  selector: 'app-view-text',
  templateUrl: './text.component.html',
  styleUrls: ['./text.component.scss'],
  // changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TextComponent extends DynamicView implements OnInit {
  @Input() view: TextView;

  value: string;

  isMarkdown: boolean;

  ngOnInit(): void {
    const view = this.view;
    this.value = view.config.value;
    this.isMarkdown = view.config.isMarkdown;
  }
}
