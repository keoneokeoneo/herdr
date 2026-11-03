// Preloaded with `node -r` by gen-golden.sh so statusline.mjs sees a fixed
// clock (it calls Date.now() for the rate-limit countdown).
const fixed = Number(process.env.FAKE_NOW_MS);
if (Number.isFinite(fixed)) Date.now = () => fixed;
