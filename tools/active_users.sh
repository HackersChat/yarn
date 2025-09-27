#!/bin/bash

URL="${URL:-"https://twtxt.net"}"

DAYS="30"
if [ $# -eq 1 ]; then
  DAYS="${1}"
fi

bat "$URL/api/v1/debug/db" "Token:$YARND_TOKEN" | jq --arg days "$DAYS" '. | map_values(@base64d) | {Key: .key, Value: .value | fromjson} | select(.Key | startswith("/users/")) | select(.Value.LastSeenAt != null) | select((now - (.Value.LastSeenAt | fromdate)) < (($days|tonumber)*24*3600)) | {Username: .Value.Username, LastSeenAt: .Value.LastSeenAt | fromdate}' | jq -s '. | sort_by(.LastSeenAt) | .[] | "\(.Username) last seen \((now - (.LastSeenAt)) / (24*3600) | ceil) days ago"'
