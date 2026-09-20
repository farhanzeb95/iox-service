package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the Supabase (PostgreSQL) connection pool
var Pool *pgxpool.Pool

// Connect establishes a connection to Supabase/PostgreSQL using the given connection string.
// Example: postgresql://postgres.[ref]:[password]@aws-0-[region].pooler.supabase.com:6543/postgres
func Connect(connString string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal("Parse Supabase connection string failed:", err)
	}
	// Supabase's transaction pooler can route queries to different backend
	// connections, so pgx prepared-statement caching is not safe here.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	config.MaxConns = 25
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Supabase connect failed:", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Supabase ping failed:", err)
	}

	Pool = pool
	log.Println("✅ Supabase (PostgreSQL) connected")
}
