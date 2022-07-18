#!/bin/sh

SERVICE_EXEC_PATH="/var/packages/mielesolar/target/bin/mielesolar"
CONFIG_FILE="/var/packages/mielesolar/target/mielesolar.conf"
LOG_FILE="/var/packages/mielesolar/target/mielesolar.log"

# Import config file
# shellcheck disable=SC1090
. "${CONFIG_FILE}"

exec "$SERVICE_EXEC_PATH" \
  -interval "${POLL_INTERVAL}" \
  -config "${CONFIG_FILE}" \
  -auto "${AUTO_POWER}" \
  -vg "${COUNTRY_SELECTOR}" > "${LOG_FILE}" 2>&1
