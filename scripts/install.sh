#!/usr/bin/env bash

set -euo pipefail

repo="sandbox0-ai/sandnote"

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

choose_install_dir() {
  if [ -n "${INSTALL_DIR:-}" ]; then
    echo "${INSTALL_DIR}"
    return
  fi

  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
  else
    echo "${HOME}/.local/bin"
  fi
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar
need_cmd mktemp

resolve_version() {
  if [ -n "${SANDNOTE_VERSION:-}" ]; then
    echo "${SANDNOTE_VERSION}"
    return
  fi

  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n1
}

os="$(detect_os)"
arch="$(detect_arch)"
install_dir="$(choose_install_dir)"
version="$(resolve_version)"

if [ -z "${version}" ]; then
  echo "failed to resolve latest sandnote release version" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

archive="sandnote_${version}_${os}_${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/${version}/${archive}"

mkdir -p "${install_dir}"

curl -fsSL "${url}" -o "${tmpdir}/${archive}"
tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"
install -m 0755 "${tmpdir}/sandnote" "${install_dir}/sandnote"

echo "installed sandnote to ${install_dir}/sandnote"

case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *)
    echo "warning: ${install_dir} is not on PATH" >&2
    ;;
esac
