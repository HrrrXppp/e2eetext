// Package metrics publishes this node's WebSocket connection count to
// CloudWatch. Dataset-wide counts (messages, chats, users) are published by
// the scheduled Lambda in lambdas/metrics, not by the server.
package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	// MetricWebSocketConnections is the CloudWatch metric name; the cluster
	// total is SUM across the NodeId dimension.
	MetricWebSocketConnections = "WebSocketConnections"

	// DefaultInterval is the publish resolution (1 minute).
	DefaultInterval = time.Minute

	publishTimeout = 10 * time.Second
)

// PutMetricDataAPI is the subset of the CloudWatch client the publisher uses.
type PutMetricDataAPI interface {
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

// Publisher periodically publishes the node's WebSocket connection count.
type Publisher struct {
	client      PutMetricDataAPI
	namespace   string
	environment string
	nodeID      string
	count       func() int
	interval    time.Duration
	logger      *slog.Logger
}

// NewPublisher builds a Publisher. count returns the current number of
// WebSocket connections on this node. environment may be empty, in which case
// the Environment dimension is omitted.
func NewPublisher(client PutMetricDataAPI, namespace, environment, nodeID string, count func() int, interval time.Duration, logger *slog.Logger) *Publisher {
	if interval <= 0 {
		interval = DefaultInterval
	}
	return &Publisher{
		client:      client,
		namespace:   namespace,
		environment: environment,
		nodeID:      nodeID,
		count:       count,
		interval:    interval,
		logger:      logger,
	}
}

// Run publishes immediately and then every interval until ctx is cancelled.
func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		p.publish(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Publisher) publish(ctx context.Context) {
	callCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if _, err := p.client.PutMetricData(callCtx, p.input(time.Now())); err != nil && ctx.Err() == nil {
		p.logger.Warn("publish websocket metric", "err", err)
	}
}

func (p *Publisher) input(now time.Time) *cloudwatch.PutMetricDataInput {
	dims := []types.Dimension{{Name: aws.String("NodeId"), Value: aws.String(p.nodeID)}}
	if p.environment != "" {
		dims = append([]types.Dimension{{Name: aws.String("Environment"), Value: aws.String(p.environment)}}, dims...)
	}
	return &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(p.namespace),
		MetricData: []types.MetricDatum{{
			MetricName: aws.String(MetricWebSocketConnections),
			Dimensions: dims,
			Timestamp:  aws.Time(now),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(float64(p.count())),
		}},
	}
}
