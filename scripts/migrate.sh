#!/bin/bash
# migrate.sh - Run SQL migrations against MySQL
set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-hls_streamer}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-}"

MIGRATIONS_DIR="$(dirname "$0")/../migrations"

echo "Running migrations on ${DB_HOST}:${DB_PORT}/${DB_NAME}..."

for f in "$MIGRATIONS_DIR"/*.sql; do
    echo "  Applying $(basename "$f")..."
    mysql -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" ${DB_PASS:+-p"$DB_PASS"} "$DB_NAME" < "$f"
done

echo "Migrations complete."
