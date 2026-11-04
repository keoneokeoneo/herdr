import { View, div } from "gpui-kit";
import { v_flex, h_flex, Input, InputState, TextView } from "gpui-base";
import { exec, appHome, theme, shared, setShared, run, nextCommand, props } from "herdr";

import { treeRows, parentFolders, displayName } from "./tree.js";

const button = (id, label, click) => div().id(id).px(10).py(6).rounded(6).cursor_pointer()
  .bg(theme().raised).hover(s => s.bg(theme().hover)).child(label).on_click((_e, cx) => click(cx));
// Wiki links become local navigation links; embeds are shown as links in this reader.
export function markdown(text) {
  return text.split(/(```[\s\S]*?```|~~~[\s\S]*?~~~|`[^`\n]*`)/g).map((part, i) => i % 2 ? part :
    part.replace(/!?\[\[([^\]\n]+)\]\]/g, (_m, body) => {
      const [target, alias] = body.split('|');
      return `[${(alias || target).replace(/[\[\]]/g, '')}](vault:${encodeURIComponent(target)})`;
    })).join('');
}

export default class Obsidian extends View {
  init(_props, cx) {
    this.embedded = !!props()?.embedded;
    this.root = shared('obsidian.root') || '';
    this.vaults = shared('obsidian.vaults') || [];
    this.expanded = new Set(); this.searching = false;
    this.rows = []; this.history = []; this.path = ''; this.text = ''; this.error = '';
    this.generation = 0; this.searchRequest = 0; this.readRequest = 0;
    this.choosing = !this.root;
    this.pathInput = InputState.new({ placeholder: '~/Documents/personal-vault', value: this.root });
    this.query = InputState.new({ placeholder: 'Search titles and text · Enter' });
    this.query.on('submit', (_e, cx) => this.search(cx));
    this.pathInput.on('submit', (_e, cx) => this.select(this.pathInput.value(), cx));
    cx.spawn(async cx => {
      try { this.vaults = [...new Set([...this.vaults, ...await this.call('vaults')])]; }
      catch (e) { this.error = String(e); }
      cx.notify();
    });
    cx.spawn(async cx => {
      for (;;) {
        const command = await nextCommand();
        if (command === 'focus') this.query.focus();
        else if (command === 'search') this.search(cx);
        else if (command.startsWith('vault:')) this.select(command.slice(6), cx);
        else if (command.startsWith('note:')) this.open(command.slice(5), cx);
        cx.notify();
      }
    });
    if (this.root) this.search(cx);
  }
  async call(action, ...args) {
    const r = await exec('python3', [appHome() + '/mods/obsidian/vault.py', action, ...args]);
    if (r.code !== 0) throw new Error(r.stderr.trim() || 'Vault를 읽을 수 없습니다.');
    return JSON.parse(r.stdout);
  }
  select(root, cx) {
    this.root = root.trim(); this.generation++; this.readRequest++; this.reading = false; this.path = ''; this.text = ''; this.history = []; this.rows = [];
    this.expanded = new Set(); this.query.set_value(''); this.search(cx, true);
  }
  search(cx, selecting = false) {
    const root = this.root, generation = this.generation, request = ++this.searchRequest;
    if (!root) return;
    const query = this.query.value();
    this.busy = true; this.error = ''; cx.notify();
    cx.spawn(async cx => {
      try {
        const result = await this.call('search', root, query);
        if (generation !== this.generation || request !== this.searchRequest) return;
        this.rows = result.notes; this.searching = !!query.trim();
        if (this.searching) for (const path of this.rows) for (const folder of parentFolders(path)) this.expanded.add(folder); this.total = result.total; this.limited = result.limited;
        if (selecting) {
          this.root = result.root; this.pathInput.set_value(result.root); this.choosing = false;
          this.vaults = [...new Set([...this.vaults, result.root])];
          setShared('obsidian.root', result.root); setShared('obsidian.vaults', this.vaults);
        }
      } catch (e) { if (generation === this.generation && request === this.searchRequest) this.error = String(e); }
      finally { if (generation === this.generation && request === this.searchRequest) this.busy = false; cx.notify(); }
    });
  }
  open(target, cx, back = false) {
    const generation = this.generation, request = ++this.readRequest, previous = this.path;
    this.error = ''; this.reading = true; cx.notify();
    cx.spawn(async cx => {
      try {
        const result = await this.call('read', this.root, target, this.path);
        if (generation !== this.generation || request !== this.readRequest) return;
        if (back) this.history.pop();
        else if (previous && previous !== result.path) this.history.push(previous);
        for (const folder of parentFolders(result.path)) this.expanded.add(folder);
        this.path = result.path; this.text = markdown(result.text);
      } catch (e) { if (generation === this.generation && request === this.readRequest) this.error = String(e); }
      finally { if (request === this.readRequest) this.reading = false; cx.notify(); }
    });
  }
  link(url, cx) {
    if (url.startsWith('vault:')) this.open(decodeURIComponent(url.slice(6)), cx);
    else if (/^https?:\/\//i.test(url)) cx.spawn(async cx => {
      try { await exec('open', [url]); } catch (e) { this.error = String(e); cx.notify(); }
    });
    else if (!/^[a-z][a-z0-9+.-]*:/i.test(url)) this.open(url, cx);
  }
  render(cx) {
    const t = theme();
    const root = v_flex().size_full().min_w(0).bg(t.surface).text_color(t.foreground).text_size(13).p(12).gap(10);
    root.child(h_flex().items_center().justify_between().gap(8)
      .child(div().font_semibold().truncate().child(displayName(this.root) || 'Choose a vault'))
      .child(h_flex().gap(6).child(button('vault-picker', 'Vault', cx => { this.choosing = !this.choosing; cx.notify(); }))
        .when(this.embedded, el => el.child(button('notes-close', '×', () => run('layout.hide', { id: 'poke-panel' }))))));
    if (this.choosing) {
      root.child(v_flex().h(Math.min(180, this.vaults.length * 36)).flex_none().overflow_y_scrollbar().gap(4).children(this.vaults.map((v, i) =>
        button(`vault-${i}`, displayName(v), cx => this.select(v, cx)))));
      root.child(Input.new(this.pathInput));
      root.child(button('vault-add', 'Use folder', cx => this.select(this.pathInput.value(), cx)));
    }
    root.child(h_flex().gap(6).child(div().flex_1().min_w(0).child(Input.new(this.query)))
      .child(button('search', this.busy ? 'Searching…' : 'Search', cx => this.search(cx))));
    if (this.error) root.child(div().text_color(t.blocked).child(this.error));
    const list = v_flex().w(190).h_full().flex_none().gap(2).overflow_y_scrollbar().children(
      treeRows(this.rows, this.expanded).map(row =>
        h_flex().id(`tree-${row.folder ? 'd' : 'f'}-${row.path}`).flex_none().h(29).pl(4 + row.depth * 12).pr(4).gap(5)
          .rounded(5).cursor_pointer().bg(row.path === this.path ? t.selected : t.surface)
          .hover(s => s.bg(t.hover)).on_click((_e, cx) => {
            if (row.folder) {
              if (this.expanded.has(row.path)) this.expanded.delete(row.path); else this.expanded.add(row.path);
              cx.notify();
            } else this.open('/' + row.path, cx);
          })
          .child(div().w(12).flex_none().text_color(t.dim).child(row.folder ? (this.expanded.has(row.path) ? '▾' : '▸') : '·'))
          .child(div().flex_1().min_w(0).truncate().text_color(row.folder ? t.muted : t.foreground).child(row.name))));
    if (!this.rows.length && !this.busy) list.child(div().p(8).text_color(t.muted).child(this.searching ? 'No matching notes.' : 'No notes.'));
    const reader = v_flex().flex_1().h_full().min_w(0).gap(8);
    reader.child(h_flex().gap(6).items_center()
      .child(button('back', '←', cx => { if (this.history.length) this.open('/' + this.history[this.history.length - 1], cx, true); }))
      .child(button('refresh-note', 'Refresh', cx => { if (this.path) this.open('/' + this.path, cx); }))
      .child(div().flex_1().truncate().text_color(t.muted).child(this.reading ? 'Loading…' : displayName(this.path, true) || 'Select a note.')));
    if (this.path) reader.child(button('open-obsidian', 'Open in Obsidian ↗', cx => {
      const uri = 'obsidian://open?path=' + encodeURIComponent(this.root + '/' + this.path);
      cx.spawn(async cx => {
        try { const result = await exec('open', [uri]); if (result.code !== 0) throw new Error(result.stderr); }
        catch (e) { this.error = String(e); cx.notify(); }
      });
    }));
    if (this.path) reader.child(div().flex_1().min_h(0).child(TextView.markdown('obsidian-note-' + this.path, this.text)
      .selectable(true).scrollable(true).on_link_click((url, cx) => this.link(url, cx))));
    else reader.child(div().p(16).text_color(t.muted).child('Select a note to read. Follow internal links without leaving Herdr.'));
    root.child(h_flex().flex_1().min_h(0).gap(12).child(list).child(reader));
    root.child(h_flex().justify_between().text_size(11).text_color(t.muted)
      .child(div().child(`${this.rows.length} shown · ${this.total || 0} notes${this.limited ? ' · Refine your search' : ''} · Read only`))
      .when(!this.embedded, el => el.child(button('poke-layout', 'Back to Poke', () => run('layout.switch', { name: 'poke' })))));
    return root;
  }
}
