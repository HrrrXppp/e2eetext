// Command metrics is a scheduled AWS Lambda (EventBridge, every minute) that
// publishes dataset-wide counts (messages, chats, users) to CloudWatch. It has
// no public invocation path: only the EventBridge rule can trigger it.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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

	once sync.Once
	db   *sql.DB
	err  error
}

// open reads DATABASE_URL from Secrets Manager once per execution environment
// and caches the connection pool across invocations.
func (r *runtime) open(ctx context.Context) (*sql.DB, error) {
	r.once.Do(func() {
		out, err := r.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &r.secretID})
		if err != nil {
			r.err = fmt.Errorf("read database secret: %w", err)
			return
		}
		if out.SecretString == nil || *out.SecretString == "" {
			r.err = fmt.Errorf("database secret has no string value")
			return
		}
		db, err := sql.Open("pgx", *out.SecretString)
		if err != nil {
			r.err = fmt.Errorf("open database: %w", err)
			return
		}
		db.SetMaxOpenConns(1)
		db.SetConnMaxLifetime(5 * time.Minute)
		r.db = db
	})
	return r.db, r.err
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
