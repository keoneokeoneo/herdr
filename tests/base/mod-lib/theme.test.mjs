import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const lum = (h) => {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16) / 255)
    .map((x) => (x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};
const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);

for (const name of ['base', 'poke']) {
  test(`${name} theme: small secondary text keeps 4.5:1 on the background and chrome`, async () => {
    const t = JSON.parse(await readFile(new URL(`../../../themes/${name}.json`, import.meta.url), 'utf8'));
    for (const fg of ['muted', 'dim']) for (const bg of [t.background, t.colors.chrome]) {
      assert.ok(ratio(t.colors[fg], bg) >= 4.5, `${fg} ${t.colors[fg]} on ${bg}: ${ratio(t.colors[fg], bg).toFixed(2)}`);
    }
    assert.ok(lum(t.colors.muted) > lum(t.colors.dim), 'muted stays brighter than dim');
  });
}
