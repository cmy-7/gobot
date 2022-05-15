import * as React from "react";
import { Graph, Addon, Shape, Cell, Node } from "@antv/x6";
import ActionNode from "./shape/shape_action";
import ConditionNode from "./shape/shape_condition";
import SelectorNode from "./shape/shape_selector";
import SequenceNode from "./shape/shape_sequence";
import RootNode from "./shape/shape_root";
import LoopNode from "./shape/shape_loop";
import WaitNode from "./shape/shape_wait";
import AssertNode from "./shape/shap_assert";

import { NodeTy,IsScriptNode } from "../../../constant/node_type";
import { Button, Tooltip, Modal, Input, Badge } from 'antd';
import { ZoomInOutlined, ZoomOutOutlined, AimOutlined, UndoOutlined, CloudUploadOutlined, BugOutlined,DeleteOutlined  } from '@ant-design/icons';

import "./graph.css";
import { message } from "antd";
import PubSub from "pubsub-js";
import Topic from "../../../constant/topic";

import moment from 'moment';

const { Dnd, Stencil } = Addon;
const { Search } = Input;

// 高亮
const magnetAvailabilityHighlighter = {
  name: "stroke",
  args: {
    attrs: {
      fill: "#fff",
      stroke: "#47C769",
    },
  },
};

type Rect = {
  wratio: number,
  woffset: number,
  hratio: number,
  hoffset: number,
}

function NewStencil(graph: Graph) {

  var selectorNod = new SelectorNode()
  var seqNod = new SequenceNode()
  var condNod = new ConditionNode()
  var assertNod = new AssertNode()
  var loopNod = new LoopNode()
  var waitNod = new WaitNode()
  var title = "Components"
  var placeholder = "Search by shape name"
  var g1title = "Normal"
  var g2title = "Prefab"

  if (moment.locale() === "en") {
    selectorNod.setAttrs({ label: { text: "Selector" } });
    seqNod.setAttrs({ label: { text: "Sequence" } });
    condNod.setAttrs({ label: { text: "Condition" } });
    assertNod.setAttrs({ label: { text: "Assert" } });
    loopNod.setAttrs({ label: { text: "Loop" } });
    waitNod.setAttrs({ label: { text: "Wait" } });

  } else if (moment.locale() === "zh-cn") {
    selectorNod.setAttrs({ label: { text: "选择" } });
    seqNod.setAttrs({ label: { text: "顺序" } });
    condNod.setAttrs({ label: { text: "条件" } });
    assertNod.setAttrs({ label: { text: "断言" } });
    loopNod.setAttrs({ label: { text: "循环" } });
    waitNod.setAttrs({ label: { text: "等待" } });

    title = "组件"
    placeholder = "通过节点名进行搜索"
    g1title = "默认节点"
    g2title = "预制节点"
  }

  var stencil = new Stencil({
    title: title,
    search(nod, keyword) {
      var attr = nod.getAttrs();
      var label = attr.label.text as String;
      if (label !== null) {
        return label.toLowerCase().indexOf(keyword.toLowerCase()) !== -1;
      }

      return false;
    },
    placeholder: placeholder,
    notFoundText: "Not Found",
    target: graph,
    collapsable: true,
    stencilGraphWidth: 180,
    stencilGraphHeight: 250,
    groups: [
      {
        name: "group1",
        title: g1title,
      },
      {
        name: "group2",
        title: g2title,
      }
    ],
  });

  stencil.load([selectorNod, seqNod, condNod, assertNod, loopNod, waitNod], "group1");
  stencil.load([new ActionNode()], "group2")

  return stencil
}

export default class GraphView extends React.Component {
  graph: Graph;
  container: HTMLElement;
  dnd: any;
  stencilContainer: HTMLDivElement;

  state = {
    isModalVisible: false,
    behaviorName: "",
    platfrom: "",
    stencil: null,
    btnDebug: "Debug",
    btnStep: "Step",
    btnUpload: "Upload",
    stepCnt: 0,
  };

  rect: Rect = {
    wratio: 0.6,
    woffset: 0,
    hratio: 0.69,
    hoffset: 0,
  }

  reloadStencil() {
    this.setState({ stencil: NewStencil(this.graph) }, () => {
      if (this.state.stencil != null) {
        var stencil = this.state.stencil as Addon.Stencil
        this.stencilContainer.appendChild(stencil.container);
      }
    })

    if (moment.locale() === "en") {
      this.setState({ btnDebug: "Debug", btnStep: "Step", btnUpload: "Upload" })
    } else if (moment.locale() === "zh-cn") {
      this.setState({ btnDebug: "调试", btnStep: "步进", btnUpload: "上传" })
    }
  }

  componentDidMount() {
    // 新建画布
    const graph = new Graph({
      width: document.body.clientWidth * this.rect.wratio,
      height: document.body.clientHeight * this.rect.hratio,
      container: this.container,
      highlighting: {
        magnetAvailable: magnetAvailabilityHighlighter,
        magnetAdsorbed: {
          name: "stroke",
          args: {
            attrs: {
              fill: "#fff",
              stroke: "#31d0c6",
            },
          },
        },
      },
      snapline: {
        enabled: true,
        sharp: true,
      },
      connecting: {
        snap: true,
        allowBlank: false,
        allowLoop: false,
        allowPort: false,
        highlight: true,
        allowMulti: false,
        connector: "rounded",
        connectionPoint: "boundary",
        router: {
          name: "er",
          args: {
            direction: "V",
          },
        },
        createEdge() {
          return new Shape.Edge({
            attrs: {
              line: {
                stroke: "#a0a0a0",
                strokeWidth: 1,
                targetMarker: {
                  name: "classic",
                  size: 3,
                },
              },
            },
          });
        },
      },
      keyboard: {
        enabled: true,
      },
      grid: {
        size: 10, // 网格大小 10px
        visible: true, // 绘制网格，默认绘制 dot 类型网格
      },
      history: true,
      selecting: {
        enabled: true,
        showNodeSelectionBox: true,
      },
      scroller: {
        enabled: true,
        pageVisible: false,
        pageBreak: false,
        pannable: true,
      },
      mousewheel: {
        enabled: true,
        modifiers: ['alt', 'meta'],
      },

    });

    var root = new RootNode();
    graph.addNode(root);

    PubSub.publish(Topic.NodeAdd, [this.getNodInfo(root), true, false]);
    PubSub.publish(Topic.HistoryClean, {})

    graph.bindKey("del", () => {
      this.ClickDel()
      return false;
    });

    graph.bindKey('ctrl+z', () => {
      PubSub.publish(Topic.Undo, {})
    })

    graph.on("edge:removed", ({ edge, options }) => {
      if (!options.ui) {
        return;
      }

      console.info("edge:removed")

      this.findNode(edge.getTargetCellId(), (child) => {
        //var ts = child.removeFromParent( { deep : false } );  // options 没用？
        PubSub.publish(Topic.LinkDisconnect, [child.id, false]);
        child.getParent()?.removeChild(edge);
        //var ts = child.removeFromParent({ deep: false });
        //this.graph.addCell(ts);
      });

      //graph.removeEdge(edge.id);
    });

    graph.on("edge:connected", ({ isNew, edge }) => {
      const source = edge.getSourceNode();
      const target = edge.getTargetNode();

      if (isNew) {
        if (source !== null && target !== null) {
          edge.setZIndex(0)
          source.addChild(target);
          PubSub.publish(Topic.LinkConnect, [{ parent: source.id, child: target.id }, false]);
        }
      }
    });

    graph.on("node:click", ({ node }) => {
      PubSub.publish(Topic.NodeClick, {
        id: node.id,
        type: node.getAttrs().type,
      });
    });

    graph.on("node:added", ({ node, index, options }) => {

      let silent = false
      let build = true

      if (options.others !== undefined) {
        silent = options.others.silent
        build = options.others.build
      }

      PubSub.publish(Topic.NodeAdd, [this.getNodInfo(node), build, silent]);

    });


    graph.on("node:moved", ({ e, x, y, node, view: NodeView }) => {
      this.findNode(node.id, (nod) => {
        PubSub.publish(Topic.UpdateGraphParm, this.getNodInfo(node));
      });
    });

    graph.on("edge:mouseenter", ({ edge }) => {
      edge.addTools([
        "source-arrowhead",
        "target-arrowhead",
        {
          name: "button-remove",
          args: {
            distance: -30,
          },
        },
      ]);
    });

    graph.on("edge:mouseleave", ({ edge }) => {
      edge.removeTools();
    });

    graph.centerContent();
    this.dnd = new Dnd({
      target: graph,
      scaled: false,
      animation: true,
    });
    this.graph = graph;

    this.reloadStencil()

    PubSub.subscribe(Topic.UpdateNodeParm, (topic: string, info: any) => {
      console.info("update", info.parm.id, info.parm.alias, info.parm.ty)
      if (info.parm.ty === NodeTy.Action) {
        this.findNode(info.parm.id, (nod) => {
          nod.setAttrs({
            label: { text: info.parm.alias },
          });
        });
      } else if (info.parm.ty === NodeTy.Loop) {
        this.findNode(info.parm.id, (nod) => {
          nod.setAttrs({
            label: { text: this.getLoopLabel(info.parm.loop) },
          });
        });
      } else if (info.parm.ty === NodeTy.Wait) {
        this.findNode(info.parm.id, (nod) => {
          nod.setAttrs({
            label: { text: info.parm.wait.toString() + " ms" },
          });
        });
      }
    });

    PubSub.subscribe(Topic.FileLoadRedraw, (topic: string, treearr: Array<any>) => {
      this.graph.clearCells();
      console.info("redraw by undo")

      treearr.forEach(element => {
        this.redraw(element, false);
      });
    });

    PubSub.subscribe(Topic.FileLoadDraw, (topic: string, treearr: Array<any>) => {
      this.graph.clearCells();
      console.info("redraw by file")

      treearr.forEach(element => {
        this.redraw(element, true);
      });

      PubSub.publish(Topic.HistoryClean, {})
    });

    PubSub.subscribe(Topic.Focus, (topic: string, info: any) => {

      if (info.Cur !== "") {
        this.setState({ stepCnt: this.state.stepCnt + 1 })
        this.findNode(info.Cur, (nod) => {
          nod.setAttrs({
            body: {
              strokeWidth: 3,
            },
          });
        });
      }
      if (info.Prev !== "") {
        this.findNode(info.Prev, (nod) => {
          nod.setAttrs({
            body: {
              strokeWidth: 1,
            },
          });
        });
      }
    });

    PubSub.subscribe(Topic.EditPlaneEditCodeResize, (topic: string, w: number) => {

      let woffset = (document.body.clientWidth - w) / document.body.clientWidth
      this.rect.wratio = 1 - woffset
      this.rect.woffset = w
      this.graph.resize(document.body.clientWidth * this.rect.wratio, document.body.clientHeight * this.rect.hratio)

    })
    PubSub.subscribe(Topic.EditPlaneEditChangeResize, (topic: string, h: number) => {

      let hoffset = (document.body.clientHeight - h) / document.body.clientHeight
      this.rect.hratio = 1 - hoffset
      this.rect.hoffset = h
      this.graph.resize(document.body.clientWidth * this.rect.wratio, document.body.clientHeight * this.rect.hratio)

    })

    PubSub.subscribe(Topic.WindowResize, (topic: string, e: number) => {
      let w, h = 0
      if (this.rect.woffset !== 0) {
        let woffset = (document.body.clientWidth - this.rect.woffset) / document.body.clientWidth
        let wratio = 1 - woffset
        w = document.body.clientWidth * wratio
      } else {
        w = document.body.clientWidth * this.rect.wratio
      }

      if (this.rect.hoffset !== 0) {
        let hoffset = (document.body.clientHeight - this.rect.hoffset) / document.body.clientHeight
        let hratio = 1 - hoffset
        h = document.body.clientHeight * hratio
      } else {
        h = document.body.clientHeight * this.rect.hratio
      }

      this.graph.resize(w, h)
    })

    PubSub.subscribe(Topic.LanuageChange, () => {
      this.reloadStencil()
    })

    var agent = navigator.userAgent.toLowerCase();
    var isMac = /macintosh|mac os x/i.test(navigator.userAgent);
    if (agent.indexOf("win32") >= 0 || agent.indexOf("wow32") >= 0) {
      this.setState({ platfrom: "win" })
    }
    if (agent.indexOf("win64") >= 0 || agent.indexOf("wow64") >= 0) {
      this.setState({ platfrom: "win" })
    }
    if (isMac) {
      this.setState({ platfrom: "mac" })
    }


  }

  getLoopLabel(val: Number) {
    var tlab = "";
    if (val !== 0) {
      tlab = val.toString() + " times";
    } else {
      tlab = "endless";
    }

    return tlab;
  }

  redrawChild(parent: any, child: any, build: boolean) {
    var nod: Node;
    if (child.ty === NodeTy.Selector) {
      nod = new SelectorNode({ id: child.id });
    } else if (child.ty === NodeTy.Sequence) {
      nod = new SequenceNode({ id: child.id });
    } else if (child.ty === NodeTy.Condition) {
      nod = new ConditionNode({ id: child.id });
    } else if (child.ty === NodeTy.Action) {
      nod = new ActionNode({ id: child.id });
    } else if (child.ty === NodeTy.Loop) {
      nod = new LoopNode({ id: child.id });
    } else if (child.ty === NodeTy.Assert) {
      nod = new AssertNode({ id: child.id });
    } else if (child.ty === NodeTy.Wait) {
      nod = new WaitNode({ id: child.id });
    } else {
      message.warn("未知的节点类型" + child.ty);
      return;
    }

    nod.setPosition({
      x: child.pos.x,
      y: child.pos.y,
    });
    // this.graph.addNode(nod, { "silent": true }); 这样使用会导致浏览器卡死
    this.graph.addNode(nod, { "others": { "build": build, "silent": true } })
    //PubSub.publish(Topic.NodeAdd, this.getNodInfo(nod));

    if (parent) {
      this.graph.addEdge(
        new Shape.Edge({
          attrs: {
            line: {
              stroke: "#a0a0a0",
              strokeWidth: 1,
              targetMarker: {
                name: "classic",
                size: 3,
              },
            },
          },
          zIndex: 0,
          source: parent,
          target: nod,
        })
      );

      parent.addChild(nod);
      PubSub.publish(Topic.LinkConnect, [{ parent: parent.id, child: nod.id }, true]);
    }


    if (IsScriptNode(child.ty)) {
      nod.setAttrs({ label: { text: child.alias } })
      PubSub.publish(Topic.UpdateNodeParm, {
        parm: {
          id: nod.id,
          ty: child.ty,
          code: child.code,
          alias: child.alias,
        },
        notify: false,
      });
    } else if (child.ty === NodeTy.Loop) {
      nod.setAttrs({ label: { text: this.getLoopLabel(child.loop) } });
      PubSub.publish(Topic.UpdateNodeParm, {
        parm: {
          id: nod.id,
          ty: child.ty,
          loop: child.loop,
        },
        notify: false,
      });
    } else if (child.ty === NodeTy.Wait) {
      nod.setAttrs({ label: { text: child.wait.toString() + " ms" } });
      PubSub.publish(Topic.UpdateNodeParm, {
        parm: {
          id: nod.id,
          ty: child.ty,
          wait: child.wait,
        },
        notify: false,
      });
    } else if (child.ty === NodeTy.Sequence) {
      nod.setAttrs({ label: { text: "seq" } });
    } else if (child.ty === NodeTy.Selector) {
      nod.setAttrs({ label: { text: "sel" } });
    }

    if (child.children && child.children.length) {
      for (var i = 0; i < child.children.length; i++) {
        this.redrawChild(nod, child.children[i], build);
      }
    }
  }

  redraw(jsontree: any, build: boolean) {

    if (jsontree.ty === NodeTy.Root) {

      var root = new RootNode({ "id": jsontree.id });
      root.setPosition({
        x: jsontree.pos.x,
        y: jsontree.pos.y,
      });

      this.graph.addNode(root, { "others": { "build": build, "silent": true } });

      if (jsontree.children && jsontree.children.length) {
        for (var i = 0; i < jsontree.children.length; i++) {
          this.redrawChild(root, jsontree.children[i], build);
        }
      }

    } else {
      this.redrawChild(null, jsontree, build)
    }

  }

  setLabel(id: String, name: String) {
    var flag = false;
    this.findNode(id, (nod) => {
      flag = true;
    });

    if (!flag) {
      message.warning("没有在树中查找到该节点 " + id);
    }
  }

  fillChildInfo(child: Node, info: any) {
    var childInfo = {
      id: child.id,
      ty: child.getAttrs().type.toString(),
      pos: {
        x: child.position().x,
        y: child.position().y,
      },
      children: [],
    };
    info.children.push(childInfo);

    child.eachChild((cchild, idx) => {
      if (cchild instanceof Node) {
        this.fillChildInfo(cchild as Node, childInfo);
      }
    });
  }

  getNodInfo(nod: Node) {
    var info = {
      id: nod.id,
      ty: nod.getAttrs().type.toString(),
      pos: {
        x: nod.position().x,
        y: nod.position().y,
      },
      children: [],
    };

    nod.eachChild((child, idx) => {
      if (child instanceof Node) {
        this.fillChildInfo(child as Node, info);
      }
    });

    return info;
  }

  refStencil = (container: HTMLDivElement) => {
    this.stencilContainer = container;
  };

  refContainer = (container: HTMLDivElement) => {
    this.container = container;
  };

  findChild = (parent: Cell, id: String, callback: (nod: Cell) => void) => {
    if (parent.id === id) {
      callback(parent);
      return;
    } else {
      parent.eachChild((child, idx) => {
        this.findChild(child, id, callback);
      });
    }
  };

  findNode = (id: String, callback: (nod: Cell) => void) => {
    var nods = this.graph.getRootNodes();
    if (nods.length >= 0) {
      if (nods[0].id === id) {
        callback(nods[0]);
      } else {
        nods[0].eachChild((child, idx) => {
          this.findChild(child, id, callback);
        });
      }
    }
  };

  refreshNode = (parent: Cell, callback: (nod: Cell) => void) => {
    callback(parent)
    parent.eachChild((child, idx) => {
      this.refreshNode(child, callback)
    })
  }

  refreshNodes = (callback: (nod: Cell) => void) => {
    var nods = this.graph.getRootNodes();
    if (nods.length >= 0) {
      callback(nods[0]);
      nods[0].eachChild((child, idx) => {
        this.refreshNode(child, callback)
      })
    }
  }

  debug = () => { };

  ClickZoomIn = () => {
    this.graph.zoomTo(this.graph.zoom() * 1.2)
  }

  ClickZoomOut = () => {
    this.graph.zoomTo(this.graph.zoom() * 0.8)
  }

  ClickZoomReset = () => {
    this.graph.zoomTo(1)
  }

  ClickUndo = () => {
    PubSub.publish(Topic.Undo, {})
  }

  ClickDel = () => {
    const cells = this.graph.getSelectedCells();

      if (cells.length) {
        for (var i = 0; i < cells.length; i++) {

          if (cells[i].getAttrs().type.toString() !== NodeTy.Root) {

            if (cells[i].getParent() == null) {
              this.graph.removeCell(cells[i])
            } else {
              PubSub.publish(Topic.NodeRmv, cells[i].id);
              cells[i].getParent()?.removeChild(cells[i]);
            }
          }
        }
      }
  }

  behaviorNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ behaviorName: e.target.value })
  };

  modalHandleOk = () => {
    this.setState({ isModalVisible: false })
    if (this.state.behaviorName !== "") {
      PubSub.publish(Topic.Upload, this.state.behaviorName);
    } else {
      message.warning("please enter the file name of the behavior tree");
    }
  };

  modalHandleCancel = () => {
    this.setState({ isModalVisible: false })
  };

  ClickUpload = () => {
    this.setState({ isModalVisible: true })
  };

  ClickStep = (e: any) => {

    var val = 1
    if (e !== "") {
      val = parseInt(e, 10);
      if (isNaN(val)) { val = 1 }
    }

    PubSub.publish(Topic.Step, val);
  };

  ClickDebug = () => {
    this.setState({ stepCnt: 0 })
    PubSub.publish(Topic.Create, "");
    this.refreshNodes((nod) => {  // 
      nod.setAttrs({
        body: {
          strokeWidth: 1,
        },
      });
    })
  }

  render() {
    return (
      <div className="app">
        <div className="app-stencil" ref={this.refStencil} />
        <div className="app-content" ref={this.refContainer} />
        <div className="app-zoom">
          <Tooltip
            placement="leftTop"
            title="ZoomIn"
          >
            <Button icon={<ZoomInOutlined />} onClick={this.ClickZoomIn} />
          </Tooltip>
          <Tooltip
            placement="leftTop"
            title="Reset"
          >
            <Button icon={<AimOutlined />} onClick={this.ClickZoomReset} />
          </Tooltip>
          <Tooltip
            placement="leftTop"
            title="ZoomOut"
          >
            <Button icon={<ZoomOutOutlined />} onClick={this.ClickZoomOut} />
          </Tooltip>
          <Tooltip
            placement="leftTop"
            title="Undo [ ctrl+z ]"
          >
            <Button icon={<UndoOutlined />} onClick={this.ClickUndo} />
          </Tooltip>
          <Tooltip
          placement="leftTop"
          title="Delete [ del ]"
          >
          <Button icon={<DeleteOutlined />} onClick={this.ClickDel} />
          </Tooltip>
          <Badge
            count={this.state.stepCnt}
            style={{ backgroundColor: '#52c41a' }}
          />
        </div>

        <div className={"app-create-" + this.state.platfrom}>
          <Tooltip
            placement="topRight"
            title={"Create a bot for debugging"}
          >
            <Button icon={<BugOutlined />} size={"small"} style={{ width: 80 }} onClick={this.ClickDebug} >{this.state.btnDebug}</Button>
          </Tooltip>
        </div>
        <div className={"app-step-" + this.state.platfrom}>
          <Search placeholder="1" size={"small"} onSearch={this.ClickStep} style={{ width: 80 }} enterButton={this.state.btnStep}></Search>
        </div>
        <div className={"app-upload-" + this.state.platfrom}>
          <Tooltip
            placement="topRight"
            title={"Upload the bot to the server"}
          >
            <Button icon={<CloudUploadOutlined />} size={"small"} style={{ width: 80 }} onClick={this.ClickUpload}> {this.state.btnUpload}</Button>
          </Tooltip>
        </div>

        <Modal
          visible={this.state.isModalVisible}
          onOk={this.modalHandleOk}
          onCancel={this.modalHandleCancel}
        >
          <Input
            placeholder="input behavior file name"
            onChange={this.behaviorNameChange}
          />
        </Modal>

      </div>
    );
  }
}