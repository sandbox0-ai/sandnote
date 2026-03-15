#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

bin="$tmp_root/sandnote"
store_root="$tmp_root/.sandnote"

cd "$repo_root"

go build -o "$bin" ./cmd/sandnote

"$bin" --root "$store_root" init

"$bin" --root "$store_root" workspace create --id ws_auth --name task/auth
"$bin" --root "$store_root" topic create --id tp_auth --name auth --orientation "Start with the auth thread."
"$bin" --root "$store_root" entry create --id en_auth --subject "auth anchor" --meaning "resume auth work here"
"$bin" --root "$store_root" thread create --id th_auth --question "How should auth work continue?" --workspace ws_auth

"$bin" --root "$store_root" entry attach en_auth --thread th_auth --topic tp_auth
"$bin" --root "$store_root" workspace focus ws_auth th_auth
"$bin" --root "$store_root" workspace use ws_auth

"$bin" --root "$store_root" resume --json >/dev/null
"$bin" --root "$store_root" thread checkpoint th_auth \
  --belief "auth-flow-is-working" \
  --open-edge "promote-the-durable-auth-understanding" \
  --next-lean "promote-auth-topic" \
  --reentry-anchor en_auth \
  --json >/dev/null

"$bin" --root "$store_root" topic promote tp_auth --thread th_auth --include-supporting --json >/dev/null
"$bin" --root "$store_root" thread transition th_auth --to settled --json >/dev/null
"$bin" --root "$store_root" topic entries tp_auth --json >/dev/null
"$bin" --root "$store_root" version >/dev/null

"$bin" --root "$store_root" overview

echo "smoke ok"
