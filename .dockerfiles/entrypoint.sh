#!/bin/sh

random_string() {
  tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 64 | head -n 1
}

[ -n "${PUID}" ] && usermod -u "${PUID}" yarnd
[ -n "${PGID}" ] && groupmod -g "${PGID}" yarnd

printf "Configuring yarnd...\n"
[ -z "${DATA}" ] && DATA="/data"
[ -z "${STORE}" ] && STORE="bitcask:///data/yarn.db"
[ -z "${OPEN_REGISTRATIONS}" ] && OPEN_REGISTRATIONS=true
[ -z "${OPEN_PROFILES}" ] && OPEN_PROFILES=true
[ -z "${COOKIE_SECRET}" ] && COOKIE_SECRET="$(random_string)"
[ -z "${MAGICLINK_SECRET}" ] && MAGICLINK_SECRET="$(random_string)"
[ -z "${API_SIGNING_KEY}" ] && API_SIGNING_KEY="$(random_string)"
export DATA STORE OPEN_REGISTRATIONS OPEN_PROFILES COOKIE_SECRET MAGICLINK_SECRET API_SIGNING_KEY

printf "Switching UID=%s and GID=%s\n" "${PUID}" "${PGID}"
exec su-exec yarnd:yarnd "$@"
