#!/usr/bin/env bash
set -e

# Default config path (inside official image)
REDIS_CONFIG_FILE="/usr/local/etc/redis/redis.conf"

# If there's a config file, copy it so we can append to it.
# Otherwise, create a new file from scratch.
if [ -f "$REDIS_CONFIG_FILE" ]; then
  cp "$REDIS_CONFIG_FILE" /tmp/redis.conf
else
  touch /tmp/redis.conf
fi


# 1) Redis Password
if [ -n "${REDIS_PASSWORD}" ]; then
  echo "requirepass ${REDIS_PASSWORD}" >> /tmp/redis.conf
fi

# 2) Append-only file (AOF) mode
if [ -n "${REDIS_APPENDONLY}" ]; then
  echo "appendonly ${REDIS_APPENDONLY}" >> /tmp/redis.conf
fi

# 3) Max memory
if [ -n "${REDIS_MAXMEMORY}" ]; then
  echo "maxmemory ${REDIS_MAXMEMORY}" >> /tmp/redis.conf
  echo "maxmemory-policy allkeys-lru" >> /tmp/redis.conf
fi

# 4) Save intervals (RDB snapshots) - example
#   e.g., "60 1" means: save if >=1 key changed in 60 seconds
if [ -n "${REDIS_SAVE_INTERVAl}" ]; then
  # expects something like "60 1"
  echo "save ${REDIS_SAVE_INTERVAL}" >> /tmp/redis.conf
fi

exec "$@" --config-file /tmp/redis.conf
