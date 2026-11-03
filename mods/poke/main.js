// One mod, many regions: each layout block passes props.view.
import { View, div } from "gpui-kit";
import { props, nextCommand } from "herdr";
import * as tabs from "./views/tabs.js";
import * as spaces from "./views/spaces.js";
import * as agentsView from "./views/agents.js";
import * as companion from "./views/companion.js";
import * as pane from "./views/pane.js";
import * as panel from "./views/panel.js";
import * as status from "./views/status.js";

const VIEWS = { tabs, spaces, agents: agentsView, companion, pane, panel, status };

export default class Poke extends View {
  init(_props, cx) {
    this.view = VIEWS[props()?.view] ?? null;
    this.state = {};
    this.view?.init?.(this.state, cx);
    // Commands reach every shown copy; only the status bar copy acts on them.
    if (props()?.view === "status") {
      cx.spawn(async (cx) => {
        for (;;) {
          const c = await nextCommand();
          this.view?.command?.(this.state, c, cx);
          cx.notify();
        }
      });
    }
  }

  render(cx) {
    if (!this.view) return div().child(`poke: unknown view ${props()?.view ?? "(none)"}`);
    return this.view.render(this.state, cx);
  }
}
