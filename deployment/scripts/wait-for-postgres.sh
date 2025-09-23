#!/bin/sh

set -e

port="${DB_PORT:-5432}"
timeout="60"

echo "Waiting for Postgres at $DB_HOST:$port (user: $DB_USER, password: $DB_PASS)..."

start_ts=$(date +%s)

until PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -p "$port" -c '\q' 2>/dev/null; do
  current_ts=$(date +%s)
  elapsed=$((current_ts - start_ts))

  if [ "$elapsed" -ge "$timeout" ]; then
    echo "ERROR: Postgres was unable to start up before timeout"
    exit 1
  fi

  echo "Postgres is not ready yet..."
  sleep 2
done

echo "Postgres is running"
exec "$@"