# Database Migrations

## Overview

Migrations are managed by the Go application and run automatically on startup.

## How It Works

1. **Embedded Migrations**: The actual migrations are embedded in `internal/database/database.go` as Go constants
2. **SQL Files**: The `.sql` files in this directory are for **documentation only** - they mirror the embedded migrations
3. **Tracking**: Applied migrations are tracked in the `schema_migrations` table
4. **Idempotent**: Migrations use `IF NOT EXISTS` and `ON CONFLICT` to be safely re-runnable

## Migration Files

| Version | File | Description |
|---------|------|-------------|
| 1 | `001_initial_schema.sql` | Base tables: users, stores, items, prices, etc. |
| 2 | `002_store_plans.sql` | Store optimization tables and address normalization |
| 3 | `003_us_states.sql` | All 50 US states + territories as regions |

## Adding New Migrations

1. Add a new constant in `internal/database/database.go`:
   ```go
   const migration003 = `
   -- Your SQL here
   `
   ```

2. Register it in the migrations map:
   ```go
   var migrations = map[int]string{
       1: migration001,
       2: migration002,
       3: migration003,  // Add this
   }
   ```

3. Create a corresponding `.sql` file here for documentation

## Running Migrations

Migrations run automatically when the app starts:

```bash
# Via Docker
docker-compose up

# Locally
go run cmd/server/main.go
```

## Schema Overview

```
regions
  └── users
        ├── user_sessions
        ├── shopping_lists
        │     ├── shopping_list_items
        │     └── store_plans
        │           └── store_plan_items
        └── stores
              └── store_prices
                    └── price_verifications

items
  ├── item_tags → tags
  └── store_prices

price_feed (activity log)
```

## Rollback

Currently, rollback is manual. For production, consider adding down migrations or using a tool like `golang-migrate`.
