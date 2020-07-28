FROM sc4h/alpine-s6overlay:3.12

ENV \
  PATH="/app/autoscan:${PATH}" \
  AUTOSCAN_CONFIG="/config/config.yml" \
  AUTOSCAN_DATABASE="/config/autoscan.db" \
  AUTOSCAN_LOG="/config/activity.log" \
  AUTOSCAN_VERBOSITY="0"

# Binary
COPY ["dist/autoscan_linux_amd64/autoscan", "/app/autoscan/autoscan"]

VOLUME ["/config"]