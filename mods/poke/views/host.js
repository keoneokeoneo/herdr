// The persona host: an invisible block that runs Poke's behaviours once per layout.
import { div } from "gpui-kit";
import { nextCommand } from "herdr";
import { start, command } from "../behaviours.js";

export function init(s, cx) {
  start(s, cx);
  cx.spawn(async (cx) => {
    for (;;) {
      const c = await nextCommand();
      command(s, c);
      cx.notify();
    }
  });
}

export function render() {
  return div().size_full();
}
