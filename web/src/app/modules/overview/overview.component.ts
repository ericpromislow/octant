// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import {
  ActivatedRoute,
  NavigationEnd,
  Params,
  Router,
  UrlSegment,
} from '@angular/router';
import { ContentResponse, View } from 'src/app/models/content';
import { IconService } from './services/icon.service';
import { ViewService } from './services/view/view.service';
import { combineLatest, Subject } from 'rxjs';
import { untilDestroyed } from 'ngx-take-until-destroy';
import { ContentService } from './services/content/content.service';
import { WebsocketService } from './services/websocket/websocket.service';
import { KubeContextService } from './services/kube-context/kube-context.service';
import { take } from 'rxjs/operators';
import _ from 'lodash';

interface LocationCallbackOptions {
  segments: UrlSegment[];
  params: Params;
  currentContext: string;
  fragment: string;
}

@Component({
  selector: 'app-overview',
  templateUrl: './overview.component.html',
  styleUrls: ['./overview.component.scss'],
  changeDetection: ChangeDetectionStrategy.Default,
})
export class OverviewComponent implements OnInit, OnDestroy {
  @ViewChild('scrollTarget', { static: true }) scrollTarget: ElementRef;
  hasTabs = false;
  hasReceivedContent = false;
  title: string = null;
  views: View[] = null;
  singleView: View = null;
  private previousUrl = '';
  private iconName: string;
  private defaultPath: string;
  private previousParams: Params;
  private navigateCancelSubject = new Subject<boolean>();

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private iconService: IconService,
    private viewService: ViewService,
    private contentService: ContentService,
    private websocketService: WebsocketService,
    private kubeContextService: KubeContextService
  ) {}

  ngOnInit() {
    this.withCurrentLocation(options => {
      this.handlePathChange(options.segments, options.params, false);
    });

    this.websocketService.reconnected.subscribe(() => {
      this.withCurrentLocation(options => {
        this.handlePathChange(options.segments, options.params, true);
        this.kubeContextService.select({ name: options.currentContext });
      }, true);
    });

    this.router.events.pipe(untilDestroyed(this)).subscribe(ev => {
      if (ev instanceof NavigationEnd) {
        this.navigateCancelSubject.next(true);
        console.log('navigation has ended', { ev });
      }
    });
  }

  ngOnDestroy() {
    this.resetView();
  }

  private withCurrentLocation(
    callback: (options: LocationCallbackOptions) => void,
    takeOne = false
  ) {
    let observable = combineLatest(
      this.route.url,
      this.route.queryParams,
      this.route.fragment,
      this.kubeContextService.selected()
    );

    if (takeOne) {
      observable = observable.pipe(take(1));
    }

    observable.subscribe(([segments, params, fragment, currentContext]) => {
      if (currentContext !== '') {
        callback({
          segments,
          params,
          fragment,
          currentContext,
        });
      }
    });
  }

  private handlePathChange(
    segments: UrlSegment[],
    queryParams: Params,
    force: boolean
  ) {
    const urlPath = segments.map(u => u.path).join('/');
    const currentPath = urlPath || this.defaultPath;
    if (
      force ||
      currentPath !== this.previousUrl ||
      !_.isEqual(queryParams, this.previousParams)
    ) {
      this.resetView();
      this.previousUrl = currentPath;
      this.previousParams = queryParams;
      this.scrollTarget.nativeElement.scrollTop = 0;
      this.navigateCancelSubject = new Subject<boolean>();

      const observable = this.contentService.contentFor(
        currentPath,
        queryParams,
        this.navigateCancelSubject
      );

      if (observable) {
        observable.pipe(untilDestroyed(this)).subscribe(contentResponse => {
          this.setContent(contentResponse);
        });
      }
    }
  }

  private resetView() {
    this.title = null;
    this.singleView = null;
    this.views = null;
    this.hasReceivedContent = false;
  }

  private setContent = (contentResponse: ContentResponse) => {
    const views = contentResponse.content.viewComponents;
    if (!views || views.length === 0) {
      this.hasReceivedContent = false;
      // TODO: show a loading screen here
      return;
    }

    this.hasTabs = views.length > 1;
    if (this.hasTabs) {
      this.views = views;
      this.title = this.viewService.titleAsText(contentResponse.content.title);
    } else if (views.length === 1) {
      this.views = null;
      this.singleView = views[0];
    }

    this.hasReceivedContent = true;
    this.iconName = this.iconService.load(contentResponse.content);
  };
}
