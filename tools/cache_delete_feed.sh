#!/bin/bash

URL="${URL:-"https://twtxt.net"}"
ADMIN="${ADMIN:-"admin"}"

if [ $# -ne 1 ]; then
  echo "Usage: $(basename "$0") <feed>"
  exit 1
fi

FEED="${1}"

read -r -s -p "Admin Password for $ADMIN: " PASSWORD

if ! TOKEN="$(bat "$URL/api/v1/auth" "username=$ADMIN" "password=$PASSWORD" | jq -r '.token')"; then
  echo "Authentication failed!"
  exit 1
fi

feed="$(jq -rn --arg x "$FEED" '$x|@uri')"

bat POST "$URL/api/v1/cache/delete?uri=$feed" "Token:$TOKEN"
