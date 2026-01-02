#!/bin/bash
# Script to create multiple databases in a single PostgreSQL instance
# Used for microservices database separation
set -e
set -u

function create_database() {
    local database=$1
    echo "Creating database '$database'"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
        SELECT 'CREATE DATABASE $database'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$database')\gexec
        GRANT ALL PRIVILEGES ON DATABASE $database TO $POSTGRES_USER;
EOSQL
    echo "✓ Database '$database' ready"
}

if [ -n "${POSTGRES_MULTIPLE_DATABASES:-}" ]; then
    echo "================================================"
    echo "Multiple database creation requested"
    echo "Databases: $POSTGRES_MULTIPLE_DATABASES"
    echo "================================================"

    for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
        create_database $db
    done

    echo "================================================"
    echo "✓ All databases created successfully"
    echo "================================================"
else
    echo "No POSTGRES_MULTIPLE_DATABASES variable set"
    echo "Using default database: $POSTGRES_DB"
fi
