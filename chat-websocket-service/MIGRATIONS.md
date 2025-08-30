# Database Migrations

This document describes the database migration system for the Chat WebSocket Service.

## Overview

The service uses `node-pg-migrate` to manage database migrations. Migrations are SQL files stored in the `/migrations` directory and are run automatically when the service starts.

## Migration Files

Migrations are stored in the `/migrations` directory and follow this naming convention:
- `NNN_description.sql` (e.g., `001_message_updates.sql`)

Each migration file contains SQL commands for both applying (up) and rolling back (down) changes.

Current migrations:
1. `001_message_updates.sql` - Adds message tracking columns and indexes
2. `002_store_message_integration.sql` - Integrates store messages into main messages table

## Running Migrations

Migrations run automatically when the service starts up. You can also run them manually using npm scripts:

```bash
# Run all pending migrations
npm run migrate

# Create a new migration file
npm run migrate:create migration_name

# Run specific migration(s)
npm run migrate:up [N]    # Apply N pending migrations (all if N is not specified)
npm run migrate:down [N]  # Roll back N applied migrations (1 if N is not specified)
```

## Migration Process

1. During service startup:
   - The service attempts to run any pending migrations
   - If migrations fail, the service will not start
   - Successful migrations are logged

2. For manual migrations:
   - Use `npm run migrate:create` to create new migration files
   - Write SQL commands for both up and down migrations
   - Test migrations locally before deployment

## Best Practices

1. Always include both `up` and `down` migrations
2. Use `IF EXISTS` / `IF NOT EXISTS` for idempotent migrations
3. Test migrations on a copy of production data
4. Back up the database before running migrations
5. Review migration files before deployment

## Deployment

Migrations are run automatically during deployment through the Docker Compose setup:

```yaml
chat-websocket-service:
  # ... other config ...
  command: sh -c "npm run migrate && npm start"
```

## Troubleshooting

If migrations fail:
1. Check the logs using `docker-compose logs chat-websocket-service`
2. Look for SQL syntax errors or constraint violations
3. Verify database connection settings
4. Check if migrations table exists (`public.pgmigrations`)

## Migration Status

To check migration status:
```bash
# List applied migrations
psql $DB_CONNECTION -c "SELECT * FROM pgmigrations ORDER BY run_on;"
```
