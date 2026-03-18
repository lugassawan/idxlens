"""IDXLens Python bindings for extracting structured financial data from IDX PDF reports."""

import ctypes
import json
import os
import platform

__version__ = "0.1.0"


def _lib_path():
    """Resolve the shared library path based on the platform."""
    system = platform.system().lower()
    if system == "darwin":
        name = "libidxlens.dylib"
    elif system == "windows":
        name = "libidxlens.dll"
    else:
        name = "libidxlens.so"

    return os.path.join(os.path.dirname(__file__), name)


def _load_lib():
    """Load the shared library, raising a clear error if not found."""
    path = _lib_path()
    if not os.path.exists(path):
        raise OSError(
            f"IDXLens shared library not found at {path}. "
            "Build it with: CGO_ENABLED=1 go build -buildmode=c-shared "
            "-o bindings/python/idxlens/libidxlens.so ./bindings/cgo/"
        )
    lib = ctypes.CDLL(path)

    lib.ExtractJSON.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
    lib.ExtractJSON.restype = ctypes.c_char_p

    lib.Classify.argtypes = [ctypes.c_char_p]
    lib.Classify.restype = ctypes.c_char_p

    lib.FreeString.argtypes = [ctypes.c_char_p]
    lib.FreeString.restype = None

    return lib


_lib = None


def _get_lib():
    """Return the cached library handle, loading on first access."""
    global _lib  # noqa: PLW0603
    if _lib is None:
        _lib = _load_lib()
    return _lib


def extract(pdf_path, doc_type=""):
    """Extract structured financial data from an IDX PDF report.

    Args:
        pdf_path: Path to the PDF file.
        doc_type: Optional report type (e.g. "balance-sheet", "income-statement").
                  If empty, the type is auto-detected.

    Returns:
        A dict containing the parsed financial statement.

    Raises:
        RuntimeError: If extraction fails.
        OSError: If the shared library is not found.
    """
    lib = _get_lib()
    result = lib.ExtractJSON(pdf_path.encode("utf-8"), doc_type.encode("utf-8"))
    data = json.loads(result)

    if "error" in data:
        raise RuntimeError(data["error"])

    return data


def classify(pdf_path):
    """Classify an IDX PDF report by type.

    Args:
        pdf_path: Path to the PDF file.

    Returns:
        A dict with keys: type, confidence, language, and optionally title.

    Raises:
        RuntimeError: If classification fails.
        OSError: If the shared library is not found.
    """
    lib = _get_lib()
    result = lib.Classify(pdf_path.encode("utf-8"))
    data = json.loads(result)

    if "error" in data:
        raise RuntimeError(data["error"])

    return data
