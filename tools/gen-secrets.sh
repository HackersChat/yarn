#!/bin/sh

random_string() {
  tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 64 | head -n 1
}

echo "      - API_SIGNING_KEY=$(random_string)"
echo "      - COOKIE_SECRET=$(random_string)"
echo "      - MAGICLINK_SECRET=$(random_string)"
