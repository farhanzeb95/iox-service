# Database migrations (Supabase / PostgreSQL)

Migrations are embedded into the Go service and run automatically at startup.
The service applies pending files in filename order and records successful files
in the `schema_migrations` table.

For a normal local or Render deployment, do not paste migration SQL manually.
Restarting or redeploying the service is safe: already-recorded migrations are
skipped.

## First deployment

The database user must be allowed to create tables, indexes, extensions, and
columns. The service runs these files in order:

1. `001_schema.sql`
2. `002_order_returns.sql`
3. `003_order_tracking.sql`
4. `004_seller_status_and_documents.sql`
5. `005_seller_store_fees.sql`
6. `006_monthly_seller_store_fees.sql`

If migrations were previously run manually, the idempotent SQL is designed to
bring the existing schema forward. The service will then record the successful
versions in `schema_migrations`.

After that, set `SUPABASE_DATABASE_URL` (or `DATABASE_URL`) in your `.env` with the project’s connection string (Settings → Database → Connection string, URI, Transaction pooler).
