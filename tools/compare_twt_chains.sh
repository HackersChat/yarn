#!/bin/bash

if [ $# -ne 3 ]; then
  echo "Usage: $(basename "$0") <source_pod> <target_pod> <conv_hash>"
  exit 1
fi

source_pod="$1"
target_pod="$2"
conv_hash="$3"

comm -3 \
  <(curl -qso - -H 'Accept: application/json' "$source_pod/conv/$conv_hash" | jq -r '.[] | .hash' | sort) \
  <(curl -qso - -H 'Accept: application/json' "$target_pod/conv/$conv_hash" | jq -r '.[] | .hash' | sort) \
  | awk '{ print $1 }'
