FROM ghcr.io/onedr0p/alpine:rolling@sha256:999384960b6114496a5e4036e945141c205d064ce23b87326bd3f8d878c5a9d4

ENV AUTOSCAN_CONFIG="/config/config.yml" \
    AUTOSCAN_DATABASE="/config/autoscan.db" \
    AUTOSCAN_LOG="/config/activity.log" \
    AUTOSCAN_VERBOSITY="0"

COPY autoscan /app/autoscan

RUN chown -R root:root /app && \
    chmod -R 755 /app && \
    rm -rf /tmp/*

USER kah

CMD ["/app/autoscan"]

EXPOSE 3030

LABEL org.opencontainers.image.source="https://github.com/aleksasiriski/autoscan"