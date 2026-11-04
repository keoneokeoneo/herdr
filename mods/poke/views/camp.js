// One shared garden. State lives on each partner, not in separate panels.
import { div } from 'gpui-kit';
import { h_flex, v_flex } from 'gpui-base';
import { agents, terminals, shared, run, openOverlay, closeOverlay, props } from 'herdr';
import { campGroups, gardenVisitors, campState } from '../lib/camp.js';
import { sprite, C } from './ui.js';

export function openCamp() {
  return openOverlay({ mod: 'poke', props: { view: 'camp' }, kind: 'window',
    key: 'poke-camp', width: 0.88, height: 0.85 });
}


const oval = (x, y, w, h, color) => div().absolute().left(x).top(y).w(w).h(h).rounded_full().bg(color);

function tree(x, y, scale = 1) {
  return div().absolute().left(x).top(y).w(100 * scale).h(110 * scale)
    .child(oval(5 * scale, 78 * scale, 92 * scale, 23 * scale, '#10271c70'))
    .child(div().absolute().left(45 * scale).top(55 * scale).w(12 * scale).h(38 * scale).rounded(4).bg('#68513a'))
    .child(oval(1 * scale, 24 * scale, 65 * scale, 58 * scale, '#244b38'))
    .child(oval(39 * scale, 20 * scale, 61 * scale, 60 * scale, '#2b5840'))
    .child(oval(20 * scale, 0, 67 * scale, 67 * scale, '#346247'))
    .child(oval(31 * scale, 8 * scale, 30 * scale, 19 * scale, '#40714f'));
}

function garden(height, mini = false) {
  const scale = mini ? 0.5 : 1;
  const ground = div().relative().w_full().min_w(mini ? 0 : 960).h(height).flex_none().overflow_hidden().bg('#203d30');
  // A continuous lawn, a stream-side path, and trees around its edges.
  ground.child(oval('5%', 55, '90%', height - 100, '#294a36'))
    .child(oval('20%', 80, '64%', height - 150, '#2d5039'))
    .child(oval('33%', height - 130, '60%', 210, '#344c36'));
  for (let i = 0; i < 15; i++) {
    const x = 7 + i * 5.9;
    const y = height - 92 - Math.sin(i * 0.42) * 33;
    ground.child(oval(`${x}%`, y, 48, 25, i % 2 ? '#77775a' : '#858366'));
  }
  ground.child(oval('76%', 24, 190 * scale, 108 * scale, '#395f4b'))
    .child(oval('78%', 34, 166 * scale, 86 * scale, '#244c53'))
    .child(oval('80%', 44, 132 * scale, 62 * scale, '#2b6065'))
    .child(oval('84%', 55, 55, 2, '#66938c'))
    .child(oval('87%', 78, 39, 2, '#66938c'))
    .child(oval('80%', 82, 19, 10, '#679565'));
  for (const [x, y, s] of [['-3%', 0, 1.3], ['8%', -35, 1.1], ['24%', -50, 1.15],
    ['43%', -44, 1], ['60%', -35, 1.2], ['93%', 122, 1.15], ['-5%', 240, 1.1],
    ['1%', height - 100, 1.2], ['88%', height - 94, 1.15]]) ground.child(tree(x, y, s * scale));
  // Small grass tufts and flowers; deterministic so redraws don't make the scenery jump.
  for (let i = 0; i < 42; i++) {
    const x = 5 + (i * 37 % 89);
    const y = 95 + (i * 83 % Math.max(1, height - 180));
    ground.child(div().absolute().left(`${x}%`).top(y).w(3).h(8).rounded_full().bg('#61805a55'));
    if (i % 3 === 0) ground.child(oval(`${x}%`, y - 3, 5, 5, ['#bbad74', '#ad8b97', '#8eacb1'][i % 9 / 3]));
  }
  // A little wooden bench by the path.
  ground.child(div().absolute().left('61%').top(height - 89).w(80).h(30).rounded(5).bg('#372e24')
    .child(div().absolute().top(0).w(80).h(10).rounded(3).bg('#a58a60'))
    .child(div().absolute().top(14).w(80).h(10).rounded(3).bg('#8e744f')));
  return ground;
}

function visitor(row, names) {
  const { label, mark, tone, detail: explanation } = campState(row.status);
  const color = C[tone];
  const species = row.partner == null ? 'Partner arriving…' : names[row.partner] ?? `#${row.partner}`;
  const detail = `${row.name} · ${row.space} · ${row.address} — ${species}: ${explanation}`;
  // Keep the sprite still: subpixel positional loops made the original art look soft.
  const face = sprite(row.partner, 96) ?? div().size(96).flex().items_center()
    .justify_center().text_size(32).text_color('#e1e7d9').child('?');
  const stage = div().relative().w(120).h(112).flex().items_center().justify_center()
    .child(oval(24, 94, 72, 14, '#10271b75')).child(face);
  let badge = div().absolute().top(0).right(0).px(8).py(2).rounded(8)
    .bg(row.status === 'blocked' ? '#eed289' : '#172c25')
    .text_size(16).font_weight(700).text_color(row.status === 'blocked' ? '#463b20' : color).child(mark);
  if (row.status === 'working') badge = badge.loop('opacity', { from: 0.65, to: 1, period: 1400 });
  stage.child(badge);
  return v_flex().id(`camp-${row.address}`).absolute().left(`${row.x}%`).top(row.y).w(184).items_center().gap(4)
    .cursor_pointer().tooltip(detail).accessibility_label(detail)
    .on_mouse_down('left', (_e, cx) => cx.stop_propagation())
    .on_click(() => run('agent.focus', { terminal: row.address }))
    .child(stage)
    .child(div().max_w(184).truncate().px(8).py(3).rounded(6).bg('#182e24')
      .text_size(14).font_weight(600).text_color('#f0f3e9').child(row.name))
    .child(div().px(7).py(2).rounded(5).bg('#182e24').text_size(13).text_color(color).child(label));
}

// The compact garden remains in the Poke tab when the expanded window closes.
export function minimap() {
  const visitors = gardenVisitors(campGroups(agents(), terminals()));
  const rows = Math.max(1, Math.ceil(visitors.length / 4));
  const height = Math.max(300, 110 + rows * 76);
  const scene = garden(height, true).id('garden-minimap').rounded(12).cursor_pointer()
    .tooltip('Open the garden for session names and details').on_click(openCamp);
  for (const [i, row] of visitors.entries()) {
    const state = campState(row.status);
    scene.child(div().absolute().left(`${8 + i % 4 * 22}%`).top(60 + Math.floor(i / 4) * 76 + (i % 2) * 8)
      .w(54).h(65).tooltip(`${row.name} · ${state.label}`)
      .child(sprite(row.partner, 48) ?? div().size(48).text_center().text_size(22).text_color('#e1e7d9').child('?'))
      .child(div().absolute().right(0).top(0).px(4).rounded(5).bg('#182e24').text_size(12)
        .text_color(C[state.tone]).child(state.mark)));
  }
  if (!visitors.length) scene.child(div().absolute().top(140).w_full().text_center().text_size(13)
    .text_color('#e1e7d9').child('Your garden is quiet.'));
  return v_flex().flex_none().gap(8).pt(14)
    .child(h_flex().items_center().gap(8)
      .child(div().text_size(13).font_weight(600).text_color('#c5d5bf').child(`Garden · ${visitors.length} sessions`))
      .child(div().flex_1())
      .child(div().id('expand-garden').cursor_pointer().text_size(12).text_color('#b6cbb0')
        .on_click(openCamp).child('Expand ↗')))
    .child(scene)
    .child(div().text_size(11).text_color('#a8bca4').child('Live view · Click to expand'));
}

export function render() {
  const groups = campGroups(agents(), terminals());
  const visitors = gardenVisitors(groups);
  const height = Math.max(620, 220 + Math.ceil(visitors.length / 4) * 200);
  const scene = garden(height);
  const names = shared('poke.names') ?? {};
  for (const row of visitors) scene.child(visitor(row, names));
  if (!visitors.length) scene.child(div().absolute().top(220).left('20%').w('60%').text_center()
    .text_size(13).text_color('#c0cdb7').child('A quiet garden. Start an agent to welcome a partner.'));
  const legend = h_flex().flex_wrap().gap(14).items_center();
  for (const [key, [label, mark, color]] of Object.keys(groups).map((key) => { const s = campState(key); return [key, [s.label, s.mark, C[s.tone]]]; })) {
    if (key === 'unknown' && !groups.unknown.length) continue;
    legend.child(div().text_size(13).text_color(color).child(`${mark}  ${label} ${groups[key].length}`));
  }
  return v_flex().size_full().bg('#1a2d25').text_color('#e1e7d9')
    .child(h_flex().flex_none().items_center().gap(10).px(20).py(12)
      .child(v_flex().gap(3)
        .child(div().text_size(22).font_weight(700).child('Poké Garden'))
        .child(div().text_size(13).text_color('#b6cbb0').child(`${visitors.length} partners · One little place to be together`)))
      .child(div().flex_1())
      .child(div().id('camp-close').px(10).py(6).rounded(8).cursor_pointer().text_size(14)
        .hover((x) => x.bg('#304a3a')).on_mouse_down('left', (_e, cx) => cx.stop_propagation())
        .on_click(() => closeOverlay(props().overlay)).child('Close ×')))
    .child(div().flex_1().min_h(0).overflow_scroll().child(scene))
    .child(h_flex().flex_none().flex_wrap().gap(12).px(20).py(10).child(legend)
      .child(div().flex_1()).child(div().text_size(12).text_color('#b6cbb0').child('Hover to meet · Click to visit')));
}
