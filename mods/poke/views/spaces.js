import { div } from "gpui-kit";
import { theme } from "herdr";

export function render(_state, _cx) {
  const t = theme();
  return div().size_full().bg(t.chrome).text_color(t.muted).text_size(12).p(8).child("spaces");
}
