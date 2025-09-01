#!/usr/bin/env bash
set -eu
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then set -o xtrace; fi

proxy_port=1080

n=270

site="https://ip.toor.sh"

function test {
    curl -s -x "socks5h://127.0.0.1:$proxy_port" --max-time 1 "$site"
}

for ((i=1; i<=n; i++)); do
  test
done
