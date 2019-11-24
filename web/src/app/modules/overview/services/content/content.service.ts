import { Injectable } from '@angular/core';
import { WebsocketService } from '../websocket/websocket.service';
import { BehaviorSubject, Observable, Observer, Subject } from 'rxjs';
import { Content, ContentResponse } from '../../../../models/content';
import { Params, Router } from '@angular/router';
import {
  Filter,
  LabelFilterService,
} from '../../../../services/label-filter/label-filter.service';
import { NamespaceService } from '../../../../services/namespace/namespace.service';
import { take } from 'rxjs/operators';

export const ContentUpdateMessage = 'content';
export const ChannelContentUpdateMessage = 'channelContent';
export const ChannelContentDestroyMessage = 'channelDestroy';

export interface ContentUpdate {
  content: Content;
  namespace: string;
  contentPath: string;
  queryParams: { [key: string]: string[] };
}

export interface ChannelContentUpdate {
  content: Content;
  namespace: string;
  contentPath: string;
  channelID: string;
  queryParams: { [key: string]: string[] };
}

export interface ChannelDestroyUpdate {
  channelID: string;
}

const emptyContentResponse: ContentResponse = {
  content: { viewComponents: [], title: [] },
};

@Injectable({
  providedIn: 'root',
})
export class ContentService {
  defaultPath = new BehaviorSubject<string>('');
  current = new BehaviorSubject<ContentResponse>(emptyContentResponse);
  channelUpdate = new Subject<ChannelContentUpdate>();
  channelDestroy = new Subject<ChannelDestroyUpdate>();

  private previousContentPath = '';

  private filters: Filter[] = [];
  get currentFilters(): Filter[] {
    return this.filters;
  }

  constructor(
    private router: Router,
    private websocketService: WebsocketService,
    private labelFilterService: LabelFilterService,
    private namespaceService: NamespaceService
  ) {
    websocketService.registerHandler(ChannelContentUpdateMessage, data => {
      const response = data as ChannelContentUpdate;

      this.channelUpdate.next(response);
      namespaceService.setNamespace(response.namespace);

      if (response.contentPath) {
        if (this.previousContentPath.length > 0) {
          if (response.contentPath !== this.previousContentPath) {
            const segments = response.contentPath.split('/');
            this.router.navigate(segments, {
              queryParams: response.queryParams,
            });
          }
        }

        this.previousContentPath = response.contentPath;
      }
    });

    websocketService.registerHandler(ChannelContentDestroyMessage, data => {
      const response = data as ChannelDestroyUpdate;
      console.log('received channel destroy', { data });
      // this.channelDestroy.next(response);
    });

    labelFilterService.filters.subscribe(filters => {
      this.filters = filters;
    });
  }

  contentFor(
    contentPath: string,
    params: Params,
    cancel: Subject<boolean>
  ): Observable<ContentResponse> | undefined {
    if (!contentPath) {
      return null;
    }
    console.log(`starting content stream for ${contentPath}`);
    const channelID = contentPath;
    let namespace =
      this.namespaceService.activeNamespace.getValue() || 'default';

    this.createContentStream(contentPath, channelID, params, namespace);

    return new Observable((observer: Observer<ContentResponse>) => {
      const updateSubscriber = this.channelUpdate.subscribe(channelUpdate => {
        if (channelUpdate.channelID !== channelID) {
          return;
        }

        observer.next(channelUpdate);
      });

      this.namespaceService.activeNamespace.subscribe(newNamespace => {
        if (namespace !== newNamespace) {
          console.log(`setting namespace to ${newNamespace} (${namespace})`);
          // this.destroyContentStream(channelID);
          this.createContentStream(
            contentPath,
            channelID,
            params,
            newNamespace
          );
          namespace = newNamespace;
        }
      });

      this.channelDestroy.subscribe(channelDestroy => {
        console.log(`channel destroyed`, { channelDestroy });
        if (updateSubscriber) {
          updateSubscriber.unsubscribe();
          observer.complete();
        }
      });

      cancel.pipe(take(1)).subscribe(_ => {
        this.destroyContentStream(channelID);
      });
    });
  }

  private createContentStream(
    contentPath: string,
    channelID: string,
    params: Params,
    namespace: string
  ) {
    const payload = { contentPath, channelID, params, namespace };
    this.websocketService.sendMessage('createContentStream', payload);
  }

  private destroyContentStream(channelID: string) {
    const destroyPayload = { channelID };
    this.websocketService.sendMessage('destroyContentStream', destroyPayload);
  }
}
