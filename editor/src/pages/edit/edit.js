import * as React from 'react';
import PubSub from "pubsub-js";

import SplitPane, { Pane } from "react-split-pane";

import GraphView from "./graph/graph";
import Edit from "./node/edit_tab";
import Blackboard from "./meta/meta";
import ChangeView from "./change/change";

import Topic from "../../constant/topic";

import "./edit.css";

export default class EditPlane extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
    };
  }

  componentDidMount() {}

  onEditChangeDrap = (e) => {
    PubSub.publish(Topic.EditPlaneEditChangeResize, e);
  }

  onEditCodeDrag = (e) => {
    PubSub.publish(Topic.EditPlaneEditCodeResize, e);
  };

  onCodeMetaDrag = (e) => {
    PubSub.publish(Topic.EditPlaneCodeMetaResize, e);
  };

  onChange = (e) =>{
    console.info("on change ", e)
  }

  render() {

    const divStyle = {
      overflow : "scroll"
    }

    return (
      <div className="container">
        <SplitPane
          split="vertical"
          defaultSize="60%"
          minSize={400}
          onDragFinished={this.onEditCodeDrag}
        >
          <SplitPane
            split="horizontal"
            defaultSize="70%"
            minSize={100}
            pane2Style= {divStyle}
            onDragFinished={this.onEditChangeDrap}
          >
            <Pane minSize={200} maxSize={1000} defaultSize="70%">
              <GraphView />
            </Pane>
            <Pane minSize={100} maxSize={700} defaultSize="30%">
              <ChangeView />
            </Pane>
          </SplitPane>

          <SplitPane
            split="horizontal"
            defaultSize={500}
            minSize={100}
            pane2Style= {divStyle}
            onDragFinished={this.onCodeMetaDrag}
          >
            <Pane minSize={200} maxSize={1000} defaultSize="50%">
              <Edit />
            </Pane>
            <Pane minSize={200} maxSize={1000} defaultSize="50%" >
              <Blackboard />
            </Pane>
          </SplitPane>
        </SplitPane>
      </div>
    );
  }
}