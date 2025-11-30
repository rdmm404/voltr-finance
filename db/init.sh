#!/bin/bash

set -e

# Construct DATABASE_URL with search_path for dbmate
export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable&search_path=transactions"

# Construct clean URL for psql (without search_path)
CLEAN_DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

echo "Waiting for database to be ready..."
dbmate wait

echo "Running database migrations..."
dbmate up

echo "Checking if database is already populated..."
POPULATED=$(psql "$CLEAN_DATABASE_URL" -f /db/data/check.sql -t -A | tr -d ' ')

if [ "$POPULATED" = "t" ]; then
    echo "Database is already populated. Skipping data execution."
else
    echo "Database is empty. Executing data files..."
    for file in /db/data/*.sql; do
        if [ -f "$file" ] && [ "$(basename "$file")" != "check.sql" ]; then
            echo "Executing $file..."
            psql "$CLEAN_DATABASE_URL" < "$file"
        fi
    done
fi

echo "Database initialization complete!"