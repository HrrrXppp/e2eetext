// Command metrics is a scheduled AWS Lambda (EventBridge, every minute) that
// publishes dataset-wide counts (messages, chats, users) to CloudWatch. It has
// no public invocation path: only the EventBridge rule can trigger it.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type runtime struct {
	namespace   string
	environment string
	secretID    string
	cw          *cloudwatch.Client
	sm          *secretsmanager.Client

	mu  sync.Mutex
	db  *sql.DB
	dsn string
}

// normalizeDatabaseURL trims the secret. A trailing newline makes pgx treat
// sslmode as invalid, and the raw string is what we compare so a secret
// update replaces the cached pool.
func normalizeDatabaseURL(secret string) (string, error) {
	dsn := strings.TrimSpace(secret)
	if dsn == "" {
		return "", fmt.Errorf("database secret has no string value")
	}
	return dsn, nil
}

// open reads DATABASE_URL from Secrets Manager on every invocation. A warm
// execution environment otherwise keeps the pool from the first read, so a
// secret change never reaches Postgres.
func (r *runtime) open(ctx context.Context) (*sql.DB, error) {
	out, err := r.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &r.secretID})
	if err != nil {
		return nil, fmt.Errorf("read database secret: %w", err)
	}
	if out.SecretString == nil {
		return nil, fmt.Errorf("database secret has no string value")
	}
	dsn, err := normalizeDatabaseURL(*out.SecretString)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db != nil && r.dsn == dsn {
		return r.db, nil
	}
	if r.db != nil {
		_ = r.db.Close()
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	r.db = db
	r.dsn = dsn
	return r.db, nil
}

func (r *runtime) handle(ctx context.Context) error {
	db, err := r.open(ctx)
	if err != nil {
		return err
	}
	return Collect(ctx, db, r.cw, r.namespace, r.environment, time.Now())
}

func main() {
	namespace := os.Getenv("METRICS_NAMESPACE")
	secretID := os.Getenv("DATABASE_URL_SECRET_ID")
	if namespace == "" || secretID == "" {
		fmt.Fprintln(os.Stderr, "METRICS_NAMESPACE and DATABASE_URL_SECRET_ID are required")
		os.Exit(1)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "load aws config:", err)
		os.Exit(1)
	}
	r := &runtime{
		namespace:   namespace,
		environment: os.Getenv("METRICS_ENVIRONMENT"),
		secretID:    secretID,
		cw:          cloudwatch.NewFromConfig(cfg),
		sm:          secretsmanager.NewFromConfig(cfg),
	}
	lambda.Start(r.handle)
}
