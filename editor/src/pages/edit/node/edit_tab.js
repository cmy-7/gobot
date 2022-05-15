import React from "react";
import {
  Tabs,
} from "antd";
import PubSub from "pubsub-js";
import Topic from "../../../constant/topic";
import ActionTab from "./edit_action";
import LoopTab from "./edit_loop";
import WaitTab from "./edit_wait";
import { NodeTy } from "../../../constant/node_type";

import moment from 'moment';
import lanMap from "../../../locales/lan";
import SequenceTab from "./edit_sequence";


const { TabPane } = Tabs;

export default class Edit extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      tab_id: "",
      tab_key: "ActionNode",
    };
  }

  clean() {
    this.setState({
      tab_id: "",
      tab_key: NodeTy.Condition,
    });
  }

  changeTab(ty, id) {
    var state = this.state;
    var tabkey = ""

    if (state.tab_id === id) {
      return;
    }

    if (ty === NodeTy.Sequence || ty === NodeTy.Selector) {
      tabkey = "other"
    } else if (ty === NodeTy.Action || ty === NodeTy.Condition || ty === NodeTy.Assert) {
      tabkey = NodeTy.Action
    } else {
      tabkey = ty
    }

    this.clean();
    this.setState({ tab_key: tabkey });
  }

  componentDidMount() {

    PubSub.subscribe(Topic.NodeClick, (topic, dat) => {
      this.changeTab(dat.type, dat.id);
      PubSub.publish(Topic.NodeEditorClick, dat);
    });
    
  }

  render() {

    return (
      <div>
        <Tabs activeKey={this.state.tab_key} size="small">
          <TabPane tab={lanMap["app.edit.tab.script"][moment.locale()]} key={NodeTy.Action} disabled={true}>
            <ActionTab />
          </TabPane>
          <TabPane tab={lanMap["app.edit.tab.loop"][moment.locale()]} key={NodeTy.Loop} disabled={true}>
            <LoopTab />
          </TabPane>
          <TabPane tab={lanMap["app.edit.tab.wait"][moment.locale()]} key={NodeTy.Wait} disabled={true}>
            <WaitTab />
          </TabPane>
          <TabPane tab={lanMap["app.edit.tab.other"][moment.locale()]} key={"other"} disabled={true}>
            <SequenceTab />
          </TabPane>
        </Tabs>
      </div>
    );
  }
}