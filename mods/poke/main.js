// Poke persona: one mod, three views chosen by props.view (companion, dex, host).
import { View, div } from "gpui-kit";
import { props } from "herdr";
import * as companion from "./views/companion.js";
import * as dex from "./views/dex.js";
import * as host from "./views/host.js";

const VIEWS = { companion, dex, host };

export default class Poke extends View {
  init(_props, cx) {
    this.view = VIEWS[props()?.view] ?? null;
    this.state = {};
    this.view?.init?.(this.state, cx);
  }

  render(cx) {
    if (!this.view) return div().child(`poke: unknown view ${props()?.view ?? "(none)"}`);
    return this.view.render(this.state, cx);
  }
}
