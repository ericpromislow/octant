import {
  Component,
  Inject,
  Input,
  OnChanges,
  OnInit,
  SimpleChanges,
} from '@angular/core';
import { GraphEdge, GraphNode, GraphView } from '../../../../models/content';
import { DOCUMENT } from '@angular/common';

@Component({
  selector: 'app-view-graph',
  templateUrl: './graph.component.html',
  styleUrls: ['./graph.component.sass'],
})
export class GraphComponent implements OnInit, OnChanges {
  @Input() view: GraphView;

  nodes: GraphNode[] = [];
  links: GraphEdge[] = [];

  constructor(@Inject(DOCUMENT) private document: Document) {}

  ngOnInit() {
    console.log('graph created');
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes.view.currentValue) {
      if (!viewIsEqual(changes.view.previousValue, changes.view.currentValue)) {
        console.log('graph updated', changes.view.currentValue);

        this.nodes = JSON.parse(
          JSON.stringify(changes.view.currentValue.config.nodes)
        );
        this.links = JSON.parse(
          JSON.stringify(changes.view.currentValue.config.links)
        );
      }
    }
  }

  clickHandler(event: any) {
    console.log(`clicked`, { event });
  }

  nodeColor(data: any) {
    const colorType = this.document.body.classList.contains('dark')
      ? 'dark'
      : 'light';

    const fg = data.palette[`${colorType}Fg`];
    const bg = data.palette[`${colorType}Bg`];

    return { fg, bg };
  }

  trackByNodeLabel(index: number, item: any) {
    return index;
  }

  nodeLabel(label: string) {
    const lineHeight = 1;

    return label.split('\n').map((line, index: number) => {
      return {
        y: `${index * lineHeight}em`,
        line,
      };
    });
  }
}

const viewIsEqual = (a?: GraphView, b: GraphView): boolean => {
  if (!a) {
    return false;
  }

  const aj = JSON.stringify(a.config);
  const bj = JSON.stringify(b.config);

  return aj === bj;
};
