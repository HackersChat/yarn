#!/bin/bash

URL="${URL:-"https://twtxt.net"}"

if [ $# -ne 1 ]; then
  echo "Usage: $(basename "$0") <feed>"
  exit 1
fi

FEED="${1}"

bat "$URL/api/v1/debug/db" "Token:$YARND_TOKEN" | jq --arg feed "$FEED" '. | map_values(@base64d) | {Key: .key, Value: .value | fromjson} | select(.Value.Following != null) | select(.Value.Following[] | values | contains($feed)) | {Username: .Value.Username, LastSeenAt: .Value.LastSeenAt | fromdate}' | jq -s --arg feed "$FEED" '. | sort_by(-.LastSeenAt) | .[] | "\(.Username) follows \($feed) and was last seen \((now - (.LastSeenAt)) / (24*3600) | ceil) days ago"'
