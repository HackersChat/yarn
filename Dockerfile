# Build
FROM golang:alpine AS build

RUN apk add --no-cache -U build-base git make ffmpeg-dev

RUN mkdir -p /src

WORKDIR /src

# Copy Makefile
COPY Makefile ./

# Install deps
RUN make deps

# Copy go.mod and go.sum and install and cache dependencies
COPY go.mod .
COPY go.sum .

# Copy default logo
COPY  ./internal/logo.svg ./internal/

# Copy static assets
COPY ./internal/theme/static/css/* ./internal/theme/static/css/
COPY ./internal/theme/static/img/* ./internal/theme/static/img/
COPY ./internal/theme/static/js/* ./internal/theme/static/js/

# Copy pages
COPY ./internal/pages/* ./internal/pages/

# Copy templates
COPY ./internal/theme/templates/* ./internal/theme/templates/

# Copy langs (localization / i18n)
COPY ./internal/langs/* ./internal/langs/

# Copy SQL Schema(s)
COPY ./internal/*.sql ./internal/

# Copy sources
COPY *.go ./
COPY ./internal/*.go ./internal/
COPY ./internal/auth/*.go ./internal/auth/
COPY ./internal/indieweb/*.go ./internal/indieweb/
COPY ./cmd/yarnd/*.go ./cmd/yarnd/

# Version/Commit (there there is no .git in Docker build context)
# NOTE: This is fairly low down in the Dockerfile instructions so
#       we don't break the Docker build cache just be changing
#       unrelated files that actually haven't changed but caused the
#       COMMIT value to change.
ARG VERSION="0.0.0"
ARG COMMIT="HEAD"
ARG BUILD=""

# Build server binary
RUN make server VERSION=$VERSION COMMIT=$COMMIT BUILD=$BUILD

# Runtime
FROM alpine:latest

RUN apk --no-cache -U add su-exec shadow ca-certificates tzdata ffmpeg

ENV PUID=1000
ENV PGID=1000

RUN addgroup -g "${PGID}" yarnd && \
    adduser -D -H -G yarnd -h /var/empty -u "${PUID}" yarnd && \
    mkdir -p /data && chown -R yarnd:yarnd /data

VOLUME /data

WORKDIR /

# force cgo resolver
ENV GODEBUG=netdns=cgo

COPY --from=build /src/yarnd /usr/local/bin/yarnd
COPY --from=build /src/internal/theme /theme

COPY .dockerfiles/entrypoint.sh /init

ENTRYPOINT ["/init"]
HEALTHCHECK --interval=10s --timeout=10s --retries=5 \
    CMD wget -nv -t1 --spider 'http://localhost:8000' || exit 1

CMD ["yarnd"]
