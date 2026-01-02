#!/bin/sh
set -e

echo "========================================="
echo "Chat WebSocket Service Starting"
echo "========================================="

# Wait for database to be ready
echo "Waiting for database..."
timeout=60
while ! nc -z postgres-db 5432; do
  if [ "$timeout" -le 0 ]; then
    echo "❌ Database connection timeout"
    exit 1
  fi
  timeout=$((timeout-1))
  sleep 1
done
echo "✓ Database connection established"

# Wait a bit more for database to be fully ready
sleep 2

# Run migrations with retry logic
echo "Running database migrations..."
MAX_RETRIES=3
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  echo "Migration attempt $((RETRY_COUNT + 1))/$MAX_RETRIES..."

  if npm run migrate; then
    echo "✓ Migrations completed successfully"
    break
  else
    RETRY_COUNT=$((RETRY_COUNT + 1))

    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
      echo "⚠️  Migration failed, retrying in 5 seconds..."
      sleep 5
    else
      echo "❌ Migration failed after $MAX_RETRIES attempts"
      echo "⚠️  Starting service anyway (migrations can be run manually)"
      echo "   Run: docker exec <container> npm run migrate"
      # Don't exit - allow service to start in degraded mode
    fi
  fi
done

echo "========================================="
echo "Starting chat websocket service..."
echo "Port: ${PORT:-8082}"
echo "Environment: ${NODE_ENV:-production}"
echo "========================================="

# Start the application
exec node src/server.js
