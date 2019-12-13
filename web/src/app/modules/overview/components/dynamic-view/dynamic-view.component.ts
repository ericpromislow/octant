/*
 * Copyright (c) 2019 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Component,
  ComponentFactoryResolver,
  ComponentRef,
  Input,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  ViewChild,
  ViewContainerRef,
} from '@angular/core';
import { View } from '../../../../models/content';
import { DynamicView } from './dynamic-view';
import { ContentSwitcherComponent } from '../content-switcher/content-switcher.component';
import { TextComponent } from '../text/text.component';
import { ListComponent } from '../list/list.component';

@Component({
  selector: 'app-dynamic-view',
  template: `
    <div #container></div>
    <pre>{{ view | json }}</pre>
  `,
  styleUrls: ['./dynamic-view.component.scss'],
})
export class DynamicViewComponent implements OnChanges, OnDestroy {
  @Input() view: View;

  @ViewChild('container', { read: ViewContainerRef, static: true })
  container: ViewContainerRef;

  private viewMappings: { [key: string]: any } = {
    text: TextComponent,
    list: ListComponent,
  };

  private componentRef: ComponentRef<{}>;

  constructor(private componentFactoryResolver: ComponentFactoryResolver) {}

  ngOnChanges(changes: SimpleChanges): void {
    const { currentValue, previousValue } = changes.view;
    if (currentValue) {
      const view = currentValue as View;
      if (view.metadata.type === 'text') {
        console.log('wtf', changes.view);
      }

      console.log(`component`, view.metadata.type);

      if (
        previousValue &&
        currentValue.metadata.checksum === previousValue.metadata.checksum
      ) {
        if (view.metadata.type === 'text') {
          console.log('previous value and the joint is equal', {
            currentValue,
            previousValue,
          });
        }
        return;
      }

      let created = false;
      if (!this.componentRef) {
        const viewType = this.getViewType(view);
        const factory = this.componentFactoryResolver.resolveComponentFactory<
          DynamicView
        >(viewType);
        this.componentRef = this.container.createComponent<DynamicView>(
          factory
        );
        created = true;
      }

      const instance = this.componentRef.instance as DynamicView;
      instance.view = view;

      if (!created) {
        this.componentRef.changeDetectorRef.detectChanges();
      }

      if (view.metadata.type === 'text') {
        console.log('dynamic: text', {
          view,
          created,
        });
      }
    }
  }

  ngOnDestroy(): void {
    if (this.componentRef) {
      this.componentRef.destroy();
    }
  }

  private getViewType(view: View) {
    const type = this.viewMappings[view.metadata.type];
    return type || ContentSwitcherComponent;
  }
}
