#!/bin/sh
set -e

# Wait for database to be ready
echo "Waiting for database..."
timeout=60
while ! nc -z postgres-db 5432; do
  if [ "$timeout" -le 0 ]; then
    echo "Database connection timeout"
    exit 1
  fi
  timeout=$((timeout-1))
  sleep 1
done

# Run migrations
echo "Running database migrations..."
npm run migrate

# Start the application
echo "Starting chat websocket service..."
exec node src/server.js
