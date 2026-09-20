package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"iox-service/database"
	router "iox-service/routers"
	"iox-service/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load the selected local environment file without overriding shell/host values.
	envFile := strings.TrimSpace(os.Getenv("ENV_FILE"))
	if envFile == "" {
		envFile = ".env"
	}
	if err := godotenv.Load(envFile); err != nil && envFile != ".env" {
		log.Fatalf("failed to load environment file %q: %v", envFile, err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "9001"
	}

	log.Println("🚀 Starting application")

	// Database: Supabase (PostgreSQL) — read from .env (loaded above)
	dbURL := os.Getenv("SUPABASE_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		log.Fatal("SUPABASE_DATABASE_URL or DATABASE_URL is required (set in .env)")
	}
	database.Connect(dbURL)
	log.Println("Database: Supabase (PostgreSQL)")

	// Initialize Supabase Storage (S3-compatible) for image uploads
	supabaseStorageEndpoint := os.Getenv("SUPABASE_STORAGE_S3_ENDPOINT")
	supabaseStorageAccessKey := os.Getenv("SUPABASE_STORAGE_ACCESS_KEY")
	supabaseStorageSecretKey := os.Getenv("SUPABASE_STORAGE_SECRET_KEY")
	supabaseStorageBucket := os.Getenv("SUPABASE_STORAGE_BUCKET")
	supabaseStoragePublicURL := os.Getenv("SUPABASE_STORAGE_PUBLIC_URL")

	if supabaseStorageEndpoint != "" && supabaseStorageAccessKey != "" && supabaseStorageSecretKey != "" {
		if supabaseStorageBucket == "" {
			supabaseStorageBucket = "products"
		}
		if supabaseStoragePublicURL == "" && strings.Contains(supabaseStorageEndpoint, ".storage.supabase.co") {
			ref := strings.Split(strings.TrimPrefix(supabaseStorageEndpoint, "https://"), ".")[0]
			supabaseStoragePublicURL = fmt.Sprintf("https://%s.supabase.co/storage/v1/object/public/%s", ref, supabaseStorageBucket)
		}
		if supabaseStoragePublicURL == "" {
			supabaseStoragePublicURL = fmt.Sprintf("%s/object/public/%s", strings.TrimSuffix(supabaseStorageEndpoint, "/storage/v1/s3"), supabaseStorageBucket)
		}
		err := services.InitSupabaseStorage(supabaseStorageEndpoint, supabaseStorageAccessKey, supabaseStorageSecretKey, supabaseStorageBucket, supabaseStoragePublicURL)
		if err != nil {
			log.Printf("⚠️  Failed to initialize Supabase Storage: %v", err)
			log.Println("   Image uploads will not be available")
		} else {
			log.Println("✅ Supabase Storage initialized")
			// Optional: register users bucket (avatars, identity documents)
			usersBucket := os.Getenv("SUPABASE_STORAGE_BUCKET_USERS")
			usersPublicURL := os.Getenv("SUPABASE_STORAGE_PUBLIC_URL_USERS")
			if usersBucket != "" {
				if usersPublicURL == "" && strings.Contains(supabaseStorageEndpoint, ".storage.supabase.co") {
					ref := strings.Split(strings.TrimPrefix(supabaseStorageEndpoint, "https://"), ".")[0]
					usersPublicURL = fmt.Sprintf("https://%s.supabase.co/storage/v1/object/public/%s", ref, usersBucket)
				}
				if usersPublicURL == "" {
					usersPublicURL = fmt.Sprintf("%s/object/public/%s", strings.TrimSuffix(supabaseStorageEndpoint, "/storage/v1/s3"), usersBucket)
				}
				if err := services.RegisterBucket(usersBucket, usersPublicURL); err != nil {
					log.Printf("⚠️  Failed to register users bucket %q: %v", usersBucket, err)
				} else {
					log.Printf("✅ Users bucket %q registered", usersBucket)
				}
			}
		}
	} else {
		log.Println("⚠️  Supabase Storage not configured - set SUPABASE_STORAGE_S3_ENDPOINT, SUPABASE_STORAGE_ACCESS_KEY, SUPABASE_STORAGE_SECRET_KEY for image uploads")
	}

	// Setup Gin router
	r := router.Setup()

	log.Printf("🟢 App is running on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
