#!/bin/sh

if [ $# -ne 1 ]; then
  printf "Usage %s <path/to/twtxt.db>\n" "$(basename "$0")"
  exit 1
fi

bitcask -p "${1}" dump | jq '. | map_values(@base64d) | {Key: .key, Value: .value | fromjson}'
