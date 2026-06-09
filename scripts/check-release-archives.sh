#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: $0 <archive.zip> [archive.zip ...]" >&2
  exit 2
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "release privacy check failed: missing command '$1'" >&2
    exit 2
  fi
}

is_example_path() {
  case "$1" in
    *.example|*.example.*|*.sample|*.sample.*|*.template|*.template.*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_blocked_path() {
  local path="$1"
  local base
  base="$(basename "$path")"

  if is_example_path "$path"; then
    return 1
  fi

  case "$path" in
    .env|*/.env|.env.*|*/.env.*|\
    auth.json|*/auth.json|cred.json|*/cred.json|big_token.json|*/big_token.json|\
    auth/*|*/auth/*|exports/*|*/exports/*|backups/*|*/backups/*|\
    token*.json|*/token*.json|*token*.json|*tokens*.json|*/codex_tokens_*.json|\
    *refresh*.json|*cpa*.json|*backup*.json|*export*.json|\
    easyllm-accounts-*.json|easyllm_accounts_*.json|\
    data/*|*/data/*|logs/*|*/logs/*|node_modules/*|*/node_modules/*|\
    *.log|*.db|*.db-*|*.sqlite|*.sqlite-*|*.sqlite3|*.sqlite3-*|\
    *.pem|*.key|*.p12|*.pfx|id_rsa|id_dsa|id_ecdsa|id_ed25519|\
    .codex/*|*/.codex/*|.claude/*|*/.claude/*|.agents/*|*/.agents/*|.cursor/*|*/.cursor/*)
      return 0
      ;;
    *)
      case "$base" in
        token*.json|*token*.json|*tokens*.json|*refresh*.json|*cpa*.json|*backup*.json|*export*.json)
          return 0
          ;;
      esac
      return 1
      ;;
  esac
}

is_placeholder_match() {
  case "$1" in
    *YOUR_*|*your_*|*example*|*sample*|*placeholder*|*dummy*|*changeme*|*"<token>"*|*"<secret>"*|*token_here*|*api_key_here*|*_here*|*"..."*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

scan_text_file() {
  local file="$1"
  local archive="$2"
  local rel="$3"
  local match
  local hit=0

  if ! grep -Iq . "$file"; then
    return 0
  fi

  while IFS= read -r match; do
    [ -z "$match" ] && continue
    if ! is_placeholder_match "$match"; then
      echo "release privacy check failed: secret-like token in $archive:$rel" >&2
      echo "  match: ${match:0:120}" >&2
      hit=1
      break
    fi
  done < <(grep -Eao 'BEGIN (RSA|DSA|EC|OPENSSH|PGP|PRIVATE) PRIVATE KEY|(^|[^[:alnum:]_])(sk-[A-Za-z0-9_-]{20,}|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AIza[0-9A-Za-z_-]{20,})([^[:alnum:]_]|$)' "$file" || true)

  if [ "$hit" -eq 1 ]; then
    return 1
  fi

  while IFS= read -r match; do
    [ -z "$match" ] && continue
    if ! is_placeholder_match "$match"; then
      echo "release privacy check failed: credential assignment in $archive:$rel" >&2
      echo "  match: ${match:0:120}" >&2
      hit=1
      break
    fi
  done < <(grep -Eao "\"?(access_token|refresh_token|id_token|api_key|proxy_api_key|cookie_token|authorization|secret_key|client_secret|password)\"?[[:space:]]*[:=][[:space:]]*[\"'][^\"'[:space:]]{16,}" "$file" || true)

  [ "$hit" -eq 0 ]
}

require_command unzip

for archive in "$@"; do
  if [ ! -f "$archive" ]; then
    echo "release privacy check failed: archive not found: $archive" >&2
    exit 2
  fi

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT

  while IFS= read -r entry; do
    [ -z "$entry" ] && continue
    case "$entry" in
      */) continue ;;
    esac
    if is_blocked_path "$entry"; then
      echo "release privacy check failed: blocked private path in $archive" >&2
      echo "  path: $entry" >&2
      exit 1
    fi
  done < <(unzip -Z -1 "$archive")

  unzip -qq "$archive" -d "$tmp_dir"
  while IFS= read -r -d '' file; do
    rel="${file#$tmp_dir/}"
    scan_text_file "$file" "$archive" "$rel"
  done < <(find "$tmp_dir" -type f -print0)

  rm -rf "$tmp_dir"
  trap - EXIT
  echo "release privacy check passed: $archive"
done
