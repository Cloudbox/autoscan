FROM ghcr.io/onedr0p/alpine:rolling@sha256:999384960b6114496a5e4036e945141c205d064ce23b87326bd3f8d878c5a9d4

ARG TARGETPLATFORM
ARG VERSION
ARG CHANNEL

ENV \
    AUTOSCAN_CONFIG="/config/config.yml" \
    AUTOSCAN_DATABASE="/config/autoscan.db" \
    AUTOSCAN_LOG="/config/activity.log" \
    AUTOSCAN_VERBOSITY="0"

#hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates
#hadolint ignore=DL3059
RUN \
    case "${TARGETPLATFORM}" in \
        'linux/amd64') export ARCH='amd64_v1' ;; \
        'linux/arm64') export ARCH='arm64' ;; \
    esac \
    && \
    curl -fsSL -o /app/autoscan \
        "https://github.com/aleksasiriski/autoscan/releases/download/v${VERSION}/autoscan_v${VERSION}_linux_${ARCH}" \
    && chown -R root:root /app \
    && chmod -R 755 /app \
    && rm -rf /tmp/*

USER kah

COPY ./apps/autoscan/entrypoint.sh /entrypoint.sh
CMD ["/entrypoint.sh"]

LABEL org.opencontainers.image.source="https://github.com/aleksasiriski/autoscan"