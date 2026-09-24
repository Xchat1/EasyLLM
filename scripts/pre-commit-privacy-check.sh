#!/usr/bin/env bash
set -euo pipefail

trim_line() {
  printf '%s' '[redacted]'
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

  if is_example_path "$path"; then
    return 1
  fi

  case "$path" in
    .env|.env.*|*/.env|*/.env.*|\
    cred.json|*/cred.json|big_token.json|*/big_token.json|\
    auth/*|*/auth/*|exports/*|*/exports/*|backups/*|*/backups/*|imports/*|*/imports/*|\
    tokens/*|*/tokens/*|accounts/*|*/accounts/*|sessions/*|*/sessions/*|cookies/*|*/cookies/*|\
    token*.json|*/token*.json|*token*.json|*tokens*.json|*/codex_tokens_*.json|\
    *refresh*.json|*cpa*.json|*backup*.json|*export*.json|*account*.json|*accounts*.json|*session*.json|*sub2api*.json|\
    easyllm-accounts-*.json|easyllm_accounts_*.json|easyllm-cursor-accounts-*.json|antigravity-accounts-*.json|\
    data/*|*/data/*|logs/*|*/logs/*|\
    *.log|*.db|*.db-*|*.sqlite|*.sqlite-*|*.sqlite3|*.sqlite3-*|*.session|*.session-data|*.cookie|*.cookies|\
    build/*|*/build/*|web/dist/*|*/web/dist/*|node_modules/*|*/node_modules/*|\
    dist/*|*/dist/*|\
    easyllm|easyllm_new|*.app|*.app/*|\
    scratch/*|*/scratch/*|.codex-go-cache/*|*/.codex-go-cache/*|总览.png|\
    .codex_tmp/*|*/.codex_tmp/*|.claude/*|*/.claude/*|.codex/*|*/.codex/*|.agents/*|*/.agents/*|.cursor/*|*/.cursor/*|\
    *.pem|*.key|*.p12|*.pfx|id_rsa|id_dsa|id_ecdsa|id_ed25519)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

should_skip_content_scan() {
  return 1
}

match_secret_line() {
	local lines="$1"
	local line

	while IFS= read -r line; do
		[ -z "$line" ] && continue

		if printf '%s\n' "$line" | grep -Eiq -- '-----BEGIN ([A-Z]+ )?PRIVATE KEY-----'; then
			printf '%s\n' "$line"
			return 0
		fi

		if printf '%s\n' "$line" | grep -Eiq '(^|[^[:alnum:]_])(sk-[A-Za-z0-9_-]{20,}|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AIza[0-9A-Za-z_-]{20,}|GOCSPX-[A-Za-z0-9_-]{20,}|eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,})([^[:alnum:]_]|$)'; then
			printf '%s\n' "$line"
			return 0
		fi

		if printf '%s\n' "$line" | grep -Eiq '(YOUR_|your_|example|sample|placeholder|dummy|changeme|<token>|<secret>|token_here|api_key_here|_here|\.{3}|json:"|"at-[A-Za-z0-9_-]+"|"rt-[A-Za-z0-9_-]+")'; then
			continue
		fi

		if printf '%s\n' "$line" | grep -Eiq '("?(access_token|refresh_token|id_token|api_key|proxy_api_key|cookie_token|authorization|secret_key|client_secret|password)"?[[:space:]]*[:=][[:space:]]*["'\''][^"'\''[:space:]]{16,})'; then
			printf '%s\n' "$line"
			return 0
		fi

		if printf '%s\n' "$line" | grep -Eiq '(^|\+)[[:space:]]*(OPENAI_API_KEY|API_KEY|ACCESS_TOKEN|REFRESH_TOKEN|ID_TOKEN|BEARER_TOKEN|COOKIE_TOKEN|PROXY_API_KEY|SECRET_KEY|CLIENT_SECRET|PASSWORD)[[:space:]]*=[[:space:]]*[^[:space:]#]{12,}'; then
			printf '%s\n' "$line"
			return 0
		fi
	done < <(printf '%s\n' "$lines" | grep -Ei 'PRIVATE KEY|sk-|github_pat_|gh[pousr]_|AIza|GOCSPX-|eyJ|access_token|refresh_token|id_token|api_key|proxy_api_key|cookie_token|authorization|secret_key|client_secret|password|bearer_token')

	return 1
}

hit_count=0
print_header=1

while IFS= read -r -d '' path; do
  [ -z "$path" ] && continue

  if is_blocked_path "$path"; then
    if [ "$print_header" -eq 1 ]; then
      echo "Commit blocked: detected staged file that commonly contains private credentials or sensitive account data."
      print_header=0
    fi
    echo "  - blocked staged file path: '$path'"
    hit_count=$((hit_count + 1))
    continue
  fi

  if should_skip_content_scan "$path"; then
    continue
  fi

  added_lines="$(git diff --cached --unified=0 --no-ext-diff -- "$path" | grep -E '^\+' | grep -vE '^\+\+\+' || true)"
  [ -z "$added_lines" ] && continue

  if preview="$(match_secret_line "$added_lines")"; then
    if [ "$print_header" -eq 1 ]; then
      echo "Commit blocked: detected staged content that looks like a secret or credential."
      print_header=0
    fi
    echo "  - suspicious content in staged file: '$path'"
    echo "    preview: $(trim_line "$preview")"
    hit_count=$((hit_count + 1))
  fi
done < <(git diff --cached --name-only -z --diff-filter=ACMRT)

if [ "$hit_count" -gt 0 ]; then
  echo
  echo "Security protection: commit aborted to prevent committing private account data."
  echo "Please unstage the sensitive file with: git reset HEAD <file>"
  exit 1
fi

exit 0
