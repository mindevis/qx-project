#!/usr/bin/env bash
# Download and install Mojang Java 17 (java-runtime-gamma) to /opt/qxsystem/java.
# Same source as QXLauncher / pkg/mojangjava — official Mojang runtime catalog.
#
# Usage (on Linux dedicated server):
#   curl -fsSL .../install-mojang-java17.sh | sudo bash
#   sudo bash infra/scripts/install-mojang-java17.sh
#   sudo DEST=/opt/qxsystem/java bash infra/scripts/install-mojang-java17.sh
#
# Requires: curl, jq, sha1sum
set -euo pipefail

CATALOG_URL="${MOJANG_CATALOG_URL:-https://piston-meta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json}"
COMPONENT="${MOJANG_JAVA_COMPONENT:-java-runtime-gamma}"
DEST="${MOJANG_JAVA_DEST:-/opt/qxsystem/java}"
FORCE="${FORCE:-0}"

die() {
  echo "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1 (install via apt/yum)"
}

maybe_sudo() {
  if [[ "$(id -u)" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

detect_platform() {
  [[ "$(uname -s)" == "Linux" ]] || die "Linux only"
  case "$(uname -m)" in
    x86_64 | amd64) echo "linux" ;;
    i386 | i486 | i586 | i686) echo "linux-i386" ;;
    aarch64 | arm64) echo "linux-arm64" ;;
    *) die "unsupported CPU: $(uname -m)" ;;
  esac
}

sha1_file() {
  sha1sum "$1" | awk '{print $1}'
}

download_file() {
  local url="$1" dest="$2" expected_sha1="${3:-}"
  local tmp dir
  tmp="$(mktemp)"
  dir="$(dirname "$dest")"
  maybe_sudo mkdir -p "$dir"

  if [[ -n "$expected_sha1" && -f "$dest" ]]; then
    if [[ "$(sha1_file "$dest")" == "$expected_sha1" ]]; then
      rm -f "$tmp"
      return 0
    fi
  fi

  curl -fsSL "$url" -o "$tmp"
  if [[ -n "$expected_sha1" ]]; then
    local got
    got="$(sha1_file "$tmp")"
    [[ "$got" == "$expected_sha1" ]] || die "sha1 mismatch for $dest (got $got, want $expected_sha1)"
  fi
  maybe_sudo install -m 644 "$tmp" "$dest"
  rm -f "$tmp"
}

main() {
  need_cmd curl
  need_cmd jq
  need_cmd sha1sum

  local platform java_bin
  platform="$(detect_platform)"
  java_bin="${DEST}/bin/java"

  if [[ "$FORCE" != "1" && -x "$java_bin" ]]; then
    if "$java_bin" -version 2>&1 | grep -qE 'version "17(\.|"|$)'; then
      echo "Java 17 already installed: $java_bin"
      "$java_bin" -version
      exit 0
    fi
  fi

  echo "Platform: $platform"
  echo "Component: $COMPONENT (Mojang Java 17)"
  echo "Destination: $DEST"

  local catalog manifest_url runtime_version
  catalog="$(curl -fsSL "$CATALOG_URL")"
  manifest_url="$(echo "$catalog" | jq -er --arg p "$platform" --arg c "$COMPONENT" \
    '.[$p][$c][0].manifest.url')"
  runtime_version="$(echo "$catalog" | jq -er --arg p "$platform" --arg c "$COMPONENT" \
    '.[$p][$c][0].version.name')"
  echo "Runtime version: $runtime_version"

  local pkg manifest_path relpath url sha1 executable count=0 total
  pkg="$(curl -fsSL "$manifest_url")"
  total="$(echo "$pkg" | jq '[.files | to_entries[] | select(.value.type == "file")] | length')"
  echo "Downloading $total files…"

  while IFS=$'\t' read -r relpath url sha1 executable; do
    [[ -n "$relpath" && -n "$url" ]] || continue
    count=$((count + 1))
    manifest_path="${DEST}/${relpath}"
    if (( count % 25 == 0 || count == 1 || count == total )); then
      echo "  [$count/$total] $relpath"
    fi
    download_file "$url" "$manifest_path" "$sha1"
    if [[ "$executable" == "true" ]]; then
      maybe_sudo chmod 755 "$manifest_path"
    fi
  done < <(
    echo "$pkg" | jq -r '
      .files | to_entries[]
      | select(.value.type == "file")
      | .key as $path
      | .value as $file
      | (
          $file.downloads.raw
          // ($file.downloads | to_entries[0].value)
        ) as $dl
      | select($dl.url != null)
      | [
          $path,
          $dl.url,
          ($dl.sha1 // ""),
          (if $file.executable then "true" else "false" end)
        ]
      | @tsv
    '
  )

  [[ -x "$java_bin" ]] || die "install finished but $java_bin is missing or not executable"

  maybe_sudo mkdir -p "$(dirname "$DEST")"
  # Ensure deploy user can run java (optional; skip if run as root during bootstrap).
  if [[ -n "${MOJANG_JAVA_OWNER:-}" ]]; then
    maybe_sudo chown -R "${MOJANG_JAVA_OWNER}:${MOJANG_JAVA_OWNER}" "$DEST"
  fi

  echo "Installed Mojang Java 17 → $java_bin"
  "$java_bin" -version
}

main "$@"
