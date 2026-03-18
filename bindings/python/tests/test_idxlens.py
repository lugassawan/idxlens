"""Basic tests for the idxlens Python bindings.

These tests verify the module can be imported and its public API is accessible.
Full integration tests require the shared library to be built first, which is
not done in CI.
"""

import importlib
import unittest


class TestImport(unittest.TestCase):
    """Verify the idxlens package can be imported."""

    def test_import_module(self):
        mod = importlib.import_module("idxlens")
        self.assertIsNotNone(mod)

    def test_version_attribute(self):
        import idxlens

        self.assertEqual(idxlens.__version__, "0.1.0")

    def test_extract_function_exists(self):
        import idxlens

        self.assertTrue(callable(idxlens.extract))

    def test_classify_function_exists(self):
        import idxlens

        self.assertTrue(callable(idxlens.classify))

    def test_extract_without_lib_raises_os_error(self):
        import idxlens

        # Reset cached lib so _load_lib runs fresh.
        idxlens._lib = None

        with self.assertRaises(OSError):
            idxlens.extract("nonexistent.pdf")

    def test_classify_without_lib_raises_os_error(self):
        import idxlens

        idxlens._lib = None

        with self.assertRaises(OSError):
            idxlens.classify("nonexistent.pdf")


if __name__ == "__main__":
    unittest.main()
