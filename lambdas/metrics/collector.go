package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// countsQuery fetches every count in a single round trip.
const countsQuery = `SELECT
	(SELECT COUNT(*) FROM messages),
	(SELECT COUNT(*) FROM chats),
	(SELECT COUNT(*) FROM users)`

// PutMetricDataAPI is the subset of the CloudWatch client the collector uses.
type PutMetricDataAPI interface {
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

// Collect counts messages, chats and users and publishes them to CloudWatch
// in a single PutMetricData call. environment may be empty, in which case the
// Environment dimension is omitted.
func Collect(ctx context.Context, db *sql.DB, cw PutMetricDataAPI, namespace, environment string, now time.Time) error {
	var dims []types.Dimension
	if environment != "" {
		dims = []types.Dimension{{Name: aws.String("Environment"), Value: aws.String(environment)}}
	}

	var messages, chats, users int64
	if err := db.QueryRowContext(ctx, countsQuery).Scan(&messages, &chats, &users); err != nil {
		return fmt.Errorf("count rows: %w", err)
	}

	datum := func(name string, n int64) types.MetricDatum {
		return types.MetricDatum{
			MetricName: aws.String(name),
			Dimensions: dims,
			Timestamp:  aws.Time(now),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(float64(n)),
		}
	}
	data := []types.MetricDatum{
		datum("Messages", messages),
		datum("Chats", chats),
		datum("Users", users),
	}

	if _, err := cw.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: data,
	}); err != nil {
		return fmt.Errorf("put metric data: %w", err)
	}
	return nil
}
