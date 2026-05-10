#!/usr/bin/env bash
set -euo pipefail

remote_name="${1:-origin}"
remote_url="${2:-}"
null_sha="0000000000000000000000000000000000000000"

trim_line() {
  printf '%s' "$1" | tr '\t' ' ' | sed 's/[[:space:]][[:space:]]*/ /g' | cut -c1-140
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
    auth/*.json|*/auth/*.json|\
    data/*|*/data/*|logs/*|*/logs/*|\
    *.log|*.db|*.db-*|*.sqlite|*.sqlite-*|*.sqlite3|*.sqlite3-*|\
    build/*|*/build/*|web/dist/*|*/web/dist/*|node_modules/*|*/node_modules/*|\
    easyllm|easyllm_new|*.app|*.app/*|\
    .codex_tmp/*|*/.codex_tmp/*|\
    *.pem|*.key|*.p12|*.pfx|id_rsa|id_dsa|id_ecdsa|id_ed25519)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

should_skip_content_scan() {
  local path="$1"
  case "$path" in
    *.md|docs/*)
      return 0
      ;;
    *)
      if is_example_path "$path"; then
        return 0
      fi
      return 1
      ;;
  esac
}

added_lines_for_file() {
  local commit="$1"
  local path="$2"

  git show --format= --unified=0 --no-ext-diff "$commit" -- "$path" \
    | grep -E '^\+' \
    | grep -vE '^\+\+\+' || true
}

match_secret_line() {
	local lines="$1"
	local line

	while IFS= read -r line; do
		[ -z "$line" ] && continue

		if printf '%s\n' "$line" | grep -Eiq 'BEGIN (RSA|DSA|EC|OPENSSH|PGP|PRIVATE) PRIVATE KEY'; then
			printf '%s\n' "$line"
			return 0
		fi

		if printf '%s\n' "$line" | grep -Eiq '(^|[^[:alnum:]_])(sk-[A-Za-z0-9_-]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AIza[0-9A-Za-z_-]{20,})([^[:alnum:]_]|$)'; then
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
	done <<< "$lines"

	return 1
}

print_header=1
hit_count=0

while read -r local_ref local_sha remote_ref remote_sha; do
  [ -z "${local_ref:-}" ] && continue
  [ "${local_sha:-$null_sha}" = "$null_sha" ] && continue

  if [ "${remote_sha:-$null_sha}" = "$null_sha" ]; then
    commit_cmd=(git rev-list "$local_sha" --not --remotes="$remote_name")
  else
    commit_cmd=(git rev-list "$remote_sha..$local_sha")
  fi

  while IFS= read -r commit; do
    [ -z "$commit" ] && continue

    while IFS= read -r -d '' path; do
      [ -z "$path" ] && continue

      if is_blocked_path "$path"; then
        if [ "$print_header" -eq 1 ]; then
          echo "Push blocked: detected files that commonly contain secrets or private data."
          echo "Remote: ${remote_name} ${remote_url}"
          echo
          print_header=0
        fi
        echo "  - commit ${commit:0:12}: blocked file path '$path'"
        hit_count=$((hit_count + 1))
        continue
      fi

      if should_skip_content_scan "$path"; then
        continue
      fi

      added_lines="$(added_lines_for_file "$commit" "$path")"
      [ -z "$added_lines" ] && continue

      if preview="$(match_secret_line "$added_lines")"; then
        if [ "$print_header" -eq 1 ]; then
          echo "Push blocked: detected content that looks like a secret or credential."
          echo "Remote: ${remote_name} ${remote_url}"
          echo
          print_header=0
        fi
        echo "  - commit ${commit:0:12}: suspicious content in '$path'"
        echo "    preview: $(trim_line "$preview")"
        hit_count=$((hit_count + 1))
      fi
    done < <(git diff-tree --root -r --no-commit-id --name-only -z --diff-filter=ACMRT "$commit")
  done < <("${commit_cmd[@]}")
done

if [ "$hit_count" -gt 0 ]; then
  echo
  echo "Fix suggestion:"
  echo "  1. Move secrets to ignored files such as .env or auth/*.json."
  echo "  2. Remove tracked secret files with: git rm --cached <file>"
  echo "  3. If secrets were committed earlier, rewrite or drop those commits before pushing."
  exit 1
fi

exit 0
