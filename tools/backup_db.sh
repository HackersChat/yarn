#!/bin/bash

URL="${URL:-"https://twtxt.net"}"
ADMIN="${ADMIN:-"admin"}"

read -r -s -p "Admin Password for $ADMIN: " PASSWORD

if ! TOKEN="$(bat "$URL/api/v1/auth" "username=$ADMIN" "password=$PASSWORD" | jq -r '.token')"; then
  echo "Authentication failed!"
  exit 1
fi

curl -qsSL -H "Token:$TOKEN" "$URL/api/v1/debug/db"
