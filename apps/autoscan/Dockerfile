FROM cgr.dev/chainguard/static:latest

ENV AUTOSCAN_CONFIG="/config/config.yml" \
    AUTOSCAN_DATABASE="/config/autoscan.db" \
    AUTOSCAN_LOG="/config/activity.log"

COPY autoscan .

CMD ["./autoscan"]

VOLUME ["/config"]

EXPOSE 3030

LABEL org.opencontainers.image.source="https://github.com/tminaorg/autoscan"
