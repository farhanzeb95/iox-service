# Database migrations (Supabase / PostgreSQL)

Run the schema once to create all tables.

1. In [Supabase Dashboard](https://app.supabase.com): open your project → **SQL Editor**.
2. Paste the contents of `001_schema.sql` and run it.

Or use the Supabase CLI / your migration tool to run `001_schema.sql` against the database.

After that, set `SUPABASE_DATABASE_URL` (or `DATABASE_URL`) in your `.env` with the project’s connection string (Settings → Database → Connection string, URI, Transaction pooler).
