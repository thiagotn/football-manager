#!/bin/sh
set -e

export PGPASSWORD=football

# Wait for postgres to be ready
until pg_isready -h postgres -U football -d football_dev; do
  echo "waiting for postgres..."
  sleep 1
done

echo "Creating schema_migrations table..."
psql -h postgres -U football -d football_dev -c "
  CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255) UNIQUE NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
" 2>/dev/null || true

echo "Running migrations..."
for f in $(ls -v /migrations/*.sql); do
  filename=$(basename "$f")

  # Check if migration already applied
  applied=$(psql -h postgres -U football -d football_dev -tc "SELECT 1 FROM schema_migrations WHERE version = '$filename' LIMIT 1" 2>/dev/null | grep -c 1 || true)

  if [ "$applied" -eq 1 ]; then
    echo "  ✓ Already applied: $filename"
  else
    echo "  → Running migration: $filename"
    psql -h postgres -U football -d football_dev -f "$f" > /dev/null 2>&1
    psql -h postgres -U football -d football_dev -c "INSERT INTO schema_migrations (version) VALUES ('$filename');" 2>/dev/null
    echo "    ✓ Complete"
  fi
done

echo "Migrations completed successfully"
