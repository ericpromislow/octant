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
  layout = 'dagre';

  constructor(@Inject(DOCUMENT) private document: Document) {}

  ngOnInit() {}

  ngOnChanges(changes: SimpleChanges): void {
    const cur = changes.view.currentValue;
    if (cur) {
      if (!viewIsEqual(changes.view.previousValue, cur)) {
        console.log('graph updated', cur);

        this.nodes = JSON.parse(JSON.stringify(cur.config.nodes));
        this.links = JSON.parse(JSON.stringify(cur.config.links)) || [];

        this.layout = cur.layout;
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

const viewIsEqual = (a: GraphView, b: GraphView): boolean => {
  if (!a) {
    return false;
  }

  const aj = JSON.stringify(a);
  const bj = JSON.stringify(b);

  return aj === bj;
};
