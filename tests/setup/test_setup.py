import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

REPO = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("setup_tool", REPO / "scripts" / "setup.py")
S = importlib.util.module_from_spec(spec)
spec.loader.exec_module(S)
MANIFEST = S.load_manifest(REPO / "scripts" / "setups.json")


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest() if Path(path).exists() else None


class SetupTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.app = Path(self.tmp.name) / "app"
        self.state = Path(self.tmp.name) / "state"
        self.app.mkdir()

    def install(self, names, **kw):
        return S.install(names, REPO, self.app, self.state, MANIFEST, dry=True, run_installers=False, **kw)

    def test_list_names_all_setups(self):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            self.assertEqual(S.main(["list"]), 0)
        for name in ("base", "obsidian", "poke"):
            self.assertIn(name, out.getvalue())

    def test_resolve_adds_requires_in_order(self):
        self.assertEqual(S.resolve(["poke"], MANIFEST), ["base", "poke"])
        self.assertEqual(S.resolve(["poke", "base", "obsidian"], MANIFEST), ["base", "poke", "obsidian"])

    def test_resolve_rejects_unknown_and_cycles(self):
        with self.assertRaises(ValueError):
            S.resolve(["nope"], MANIFEST)
        cyc = {"a": {"requires": ["b"]}, "b": {"requires": ["a"]}}
        with self.assertRaises(ValueError):
            S.resolve(["a"], cyc)

    def test_install_obsidian_only_copies_obsidian_and_base_theme(self):
        self.install(["obsidian"])
        for rel in ("mods/obsidian/main.js", "mods/obsidian/vault.py", "layouts/obsidian.json", "themes/base.json"):
            self.assertTrue((self.app / rel).exists(), rel)
        for rel in ("mods/base", "mods/poke", "layouts/base.json", "layouts/poke.json", "themes/poke.json"):
            self.assertFalse((self.app / rel).exists(), rel)

    def test_install_obsidian_leaves_zshrc_programs_settings_untouched(self):
        home = Path(self.tmp.name) / "home"
        home.mkdir()
        files = {home / ".zshrc": "alias x=y\n", self.app / "programs.json": "{}\n", self.app / "settings.json": "{}\n"}
        for p, text in files.items():
            p.write_text(text)
        before = {p: digest(p) for p in files}
        os.environ["HOME"] = str(home)
        self.install(["obsidian"])
        self.assertEqual(before, {p: digest(p) for p in files})

    def test_install_poke_pulls_in_base(self):
        summary = self.install(["poke"])
        self.assertEqual(summary["order"], ["base", "poke"])
        for rel in ("mods/base/main.js", "mods/base/fonts/OFL.txt", "mods/poke/main.js", "layouts/base.json",
                    "layouts/poke.json", "themes/base.json", "themes/poke.json"):
            self.assertTrue((self.app / rel).exists(), rel)
        self.assertFalse((self.app / "mods/base/gpui-kit.d.ts").exists())
        keys = json.loads((self.app / "keys.json").read_text())
        self.assertEqual(keys["layout:poke"]["cmd-j"], "base.jump")
        self.assertEqual(keys["layout:base"]["cmd-k"], "layout.toggle:panel")

    def test_second_install_is_noop(self):
        self.install(["poke"])
        snapshot = {p: digest(p) for p in self.app.rglob("*") if p.is_file()}
        again = self.install(["poke"])
        self.assertEqual(again["copied"], [])
        self.assertEqual(snapshot, {p: digest(p) for p in self.app.rglob("*") if p.is_file()})
        self.assertEqual([p for p in self.app.rglob("*") if "setup-backup" in p.name], [])

    def test_existing_different_binding_is_kept_and_reported(self):
        (self.app / "keys.json").write_text(json.dumps({"global": {}, "layout:base": {"cmd-j": "my.jump"}}))
        summary = self.install(["base"])
        keys = json.loads((self.app / "keys.json").read_text())
        self.assertEqual(keys["layout:base"]["cmd-j"], "my.jump")
        self.assertEqual(keys["layout:base"]["cmd-k"], "layout.toggle:panel")
        self.assertTrue(any("cmd-j" in k for k in summary["kept"]))

    def test_differing_target_file_is_backed_up_outside_the_app_home(self):
        # a write inside mods/ reloads the mod, so backups live in the state folder
        (self.app / "mods" / "base" / "views").mkdir(parents=True)
        (self.app / "mods" / "base" / "views" / "panel.js").write_text("old")
        (self.app / "keys.json").write_text(json.dumps({"global": {}}))
        self.install(["base"])
        self.assertEqual([p for p in self.app.rglob("*") if "setup-backup" in p.name], [])
        backups = list((self.state / "backups").glob("*/mods/base/views/panel.js"))
        self.assertEqual(len(backups), 1)
        self.assertEqual(backups[0].read_text(), "old")
        self.assertEqual(len(list((self.state / "backups").glob("*/keys.json"))), 1)

    def test_update_refreshes_only_installed_setups_without_installers(self):
        env = {"HERDR_APP_HOME": str(self.app), "HERDR_SETUP_STATE": str(self.state), "HERDR_SETUP_DRY": "1"}
        with patch.dict(os.environ, env):
            self.assertEqual(S.main(["update"]), 1)  # nothing installed yet
            self.install(["obsidian"])
            (self.app / "mods/obsidian/main.js").write_text("stale")
            with patch.object(S, "install", wraps=S.install) as spy:
                self.assertEqual(S.main(["update"]), 0)
            self.assertEqual(spy.call_args.args[0], ["obsidian"])
            self.assertFalse(spy.call_args.kwargs["run_installers"])
        self.assertEqual((self.app / "mods/obsidian/main.js").read_bytes(), (REPO / "mods/obsidian/main.js").read_bytes())
        self.assertFalse((self.app / "mods/base").exists())

    def test_in_place_checkout_copies_nothing(self):
        summary = S.install(["base"], REPO, REPO, self.state, MANIFEST, dry=True, run_installers=False)
        self.assertEqual(summary["copied"], [])
        self.assertEqual(json.loads((self.state / "setups.json").read_text())["installed"].keys(), {"base"})

    def test_every_manifest_path_exists_in_repo(self):
        for name in MANIFEST:
            for rel in S.setup_files(name, MANIFEST, REPO):
                self.assertTrue((REPO / rel).exists(), f"{name}: {rel}")

    def test_unknown_setup_exits_2(self):
        err = io.StringIO()
        with contextlib.redirect_stderr(err):
            self.assertEqual(S.main(["install", "nope"]), 2)


if __name__ == "__main__":
    unittest.main()
