"""Tests for check-doc-links.py. Run: python3 -m unittest scripts/test_check_doc_links.py -v"""

import importlib.util
import tempfile
import unittest
from pathlib import Path

_spec = importlib.util.spec_from_file_location(
    "check_doc_links", Path(__file__).resolve().parent / "check-doc-links.py"
)
links = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(links)


class CheckDocLinksTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def write(self, name, text):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
        return path

    def errors(self, *names):
        return links.check_files([self.root / n for n in names], self.root)

    def test_ok_relative_link(self):
        self.write("a.md", "[x](b.md)\n")
        self.write("b.md", "# B\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_missing_file(self):
        self.write("a.md", "[x](nope.md)\n")
        errs = self.errors("a.md")
        self.assertEqual(len(errs), 1)
        self.assertIn("nope.md", errs[0])

    def test_missing_anchor(self):
        self.write("a.md", "[x](b.md#nope)\n")
        self.write("b.md", "## Real heading\n")
        errs = self.errors("a.md")
        self.assertEqual(len(errs), 1)
        self.assertIn("b.md#nope", errs[0])

    def test_ok_anchor(self):
        self.write("a.md", "## Local heading\n\n[x](b.md#real-heading) [y](#local-heading)\n")
        self.write("b.md", "## Real heading\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_ignores_code_fences(self):
        self.write("a.md", "```\n[x](missing.md)\n```\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_ignores_external(self):
        self.write("a.md", "[a](https://example.com/x.md) [b](http://example.com) [c](mailto:x@example.com)\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_relative_dirs_and_images(self):
        self.write("docs/README.md", "# Docs\n")
        self.write("sub/img.png", "png")
        self.write("sub/a.md", "[x](../docs/README.md) ![i](img.png) ![j](gone.png)\n")
        errs = self.errors("sub/a.md")
        self.assertEqual(len(errs), 1)
        self.assertIn("gone.png", errs[0])

    def test_error_names_file_and_line(self):
        self.write("a.md", "ok\n\n[x](nope.md)\n")
        errs = self.errors("a.md")
        self.assertEqual(len(errs), 1)
        self.assertTrue(errs[0].startswith("a.md:3: "), errs[0])

    def test_skips_missing_input_file(self):
        # A file deleted in the commit can still be passed in by a hook.
        self.assertEqual(self.errors("deleted.md"), [])

    def test_underscores_in_headings(self):
        # GitHub keeps underscores in anchors: CGO_ENABLED, my_var_name.
        self.write("b.md", "## The `CGO_ENABLED` flag\n\n## my_var_name\n")
        self.write("a.md", "[x](b.md#the-cgo_enabled-flag) [y](b.md#my_var_name)\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_emphasis_in_headings(self):
        self.write("b.md", "## A **bold** and *italic* word\n")
        self.write("a.md", "[x](b.md#a-bold-and-italic-word)\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_duplicate_headings(self):
        self.write("b.md", "## Notes\n\n## Notes\n")
        self.write("a.md", "[x](b.md#notes) [y](b.md#notes-1) [z](b.md#notes-2)\n")
        errs = self.errors("a.md")
        self.assertEqual(len(errs), 1)
        self.assertIn("notes-2", errs[0])

    def test_ignores_inline_code(self):
        self.write("a.md", "Write `[x](missing.md)` for a link.\n")
        self.assertEqual(self.errors("a.md"), [])

    def test_select_skips_deleted_files(self):
        self.write("docs/a.md", "ok\n")
        files = links.select_markdown(["docs/gone.md", "docs/a.md"], self.root)
        self.assertEqual(files, [self.root / "docs/a.md"])


if __name__ == "__main__":
    unittest.main()
