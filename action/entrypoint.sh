#!/usr/bin/env bash
#
# IDXLens GitHub Action entrypoint.
#
# Usage:
#   entrypoint.sh install   — download or build the idxlens binary
#   entrypoint.sh extract   — run extraction on the target PDF
#
# Environment variables (set by action.yml):
#   IDXLENS_VERSION       — version tag (e.g. "1.2.0" or "latest")
#   IDXLENS_PDF_PATH      — path to the PDF file
#   IDXLENS_REPORT_TYPE   — optional report type flag
#   IDXLENS_OUTPUT_FORMAT — output format (json, csv)
#   IDXLENS_OUTPUT_PATH   — optional file path for output
#
set -euo pipefail

REPO="lugassawan/idxlens"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

die() {
  echo "::error::$*" >&2
  exit 1
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${arch}" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    arm64)   arch="arm64" ;;
    *)       die "unsupported architecture: ${arch}" ;;
  esac

  case "${os}" in
    linux|darwin) ;;
    mingw*|msys*|cygwin*) os="windows" ;;
    *) die "unsupported OS: ${os}" ;;
  esac

  echo "${os}_${arch}"
}

resolve_version() {
  local version="${IDXLENS_VERSION:-latest}"

  if [ "${version}" = "latest" ]; then
    version="$(gh release view --repo "${REPO}" --json tagName --jq '.tagName' 2>/dev/null)" \
      || die "failed to resolve latest release — check that gh CLI is authenticated"
  fi

  # Strip leading "v" if present for the download URL.
  version="${version#v}"
  echo "${version}"
}

# ---------------------------------------------------------------------------
# install — download pre-built binary, fall back to go install
# ---------------------------------------------------------------------------

cmd_install() {
  local version platform archive_name url install_dir

  version="$(resolve_version)"
  platform="$(detect_platform)"
  archive_name="idxlens_${version}_${platform}.tar.gz"
  if [ "${platform%%_*}" = "windows" ]; then
    archive_name="idxlens_${version}_${platform}.zip"
  fi
  url="https://github.com/${REPO}/releases/download/v${version}/${archive_name}"
  install_dir="${RUNNER_TEMP:-/tmp}/idxlens-bin"

  mkdir -p "${install_dir}"

  echo "Downloading IDXLens v${version} for ${platform}..."
  if curl -fsSL "${url}" -o "${install_dir}/${archive_name}" 2>/dev/null; then
    # Extract the binary.
    case "${archive_name}" in
      *.tar.gz) tar -xzf "${install_dir}/${archive_name}" -C "${install_dir}" ;;
      *.zip)    unzip -qo "${install_dir}/${archive_name}" -d "${install_dir}" ;;
    esac
    chmod +x "${install_dir}/idxlens"
    echo "${install_dir}" >> "${GITHUB_PATH}"
    echo "IDXLens v${version} installed from release archive."
    return
  fi

  echo "::warning::Release binary not found; falling back to go install."
  if ! command -v go &>/dev/null; then
    die "go toolchain not found — install Go or use a version with a published release"
  fi

  GOBIN="${install_dir}" go install "github.com/${REPO}/cmd/idxlens@v${version}" \
    || die "go install failed for v${version}"
  echo "${install_dir}" >> "${GITHUB_PATH}"
  echo "IDXLens v${version} installed via go install."
}

# ---------------------------------------------------------------------------
# extract — run the idxlens extract command
# ---------------------------------------------------------------------------

cmd_extract() {
  local pdf_path="${IDXLENS_PDF_PATH:?pdf-path input is required}"
  local format="${IDXLENS_OUTPUT_FORMAT:-json}"
  local report_type="${IDXLENS_REPORT_TYPE:-}"
  local output_path="${IDXLENS_OUTPUT_PATH:-}"

  if [ ! -f "${pdf_path}" ]; then
    die "PDF file not found: ${pdf_path}"
  fi

  local -a args=("extract" "financial" "--format" "${format}")

  if [ -n "${report_type}" ]; then
    args+=("--type" "${report_type}")
  fi

  if [ -n "${output_path}" ]; then
    args+=("--output" "${output_path}")
  fi

  args+=("${pdf_path}")

  echo "Running: idxlens ${args[*]}"

  if [ -n "${output_path}" ]; then
    idxlens "${args[@]}"
  else
    local result
    result="$(idxlens "${args[@]}")"
    echo "${result}"

    # Expose the result as a step output.
    {
      echo "result<<IDXLENS_EOF"
      echo "${result}"
      echo "IDXLENS_EOF"
    } >> "${GITHUB_OUTPUT}"
  fi
}

# ---------------------------------------------------------------------------
# Main dispatch
# ---------------------------------------------------------------------------

case "${1:-}" in
  install) cmd_install ;;
  extract) cmd_extract ;;
  *)       die "usage: entrypoint.sh {install|extract}" ;;
esac
