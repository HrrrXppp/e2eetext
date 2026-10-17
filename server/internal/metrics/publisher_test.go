package metrics

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type fakeCW struct {
	calls chan *cloudwatch.PutMetricDataInput
}

func (f *fakeCW) PutMetricData(_ context.Context, in *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	f.calls <- in
	return &cloudwatch.PutMetricDataOutput{}, nil
}

func newTestPublisher(client PutMetricDataAPI, env string, interval time.Duration) *Publisher {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewPublisher(client, "E2EEText/Messenger", env, "node-1", func() int { return 7 }, interval, logger)
}

func TestInputDimensionsAndValue(t *testing.T) {
	in := newTestPublisher(&fakeCW{}, "dev", 0).input(time.Unix(0, 0))

	if *in.Namespace != "E2EEText/Messenger" {
		t.Fatalf("namespace = %q", *in.Namespace)
	}
	d := in.MetricData[0]
	if *d.MetricName != MetricWebSocketConnections || *d.Value != 7 {
		t.Fatalf("datum = %s %v", *d.MetricName, *d.Value)
	}
	if len(d.Dimensions) != 2 || *d.Dimensions[0].Name != "Environment" || *d.Dimensions[0].Value != "dev" ||
		*d.Dimensions[1].Name != "NodeId" || *d.Dimensions[1].Value != "node-1" {
		t.Fatalf("dimensions = %+v", d.Dimensions)
	}
}

func TestInputOmitsEmptyEnvironment(t *testing.T) {
	in := newTestPublisher(&fakeCW{}, "", 0).input(time.Now())
	if len(in.MetricData[0].Dimensions) != 1 {
		t.Fatalf("dimensions = %+v", in.MetricData[0].Dimensions)
	}
}

func TestRunPublishesImmediatelyAndStopsOnCancel(t *testing.T) {
	fake := &fakeCW{calls: make(chan *cloudwatch.PutMetricDataInput, 4)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		newTestPublisher(fake, "dev", time.Hour).Run(ctx)
		close(done)
	}()

	select {
	case <-fake.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("no immediate publish")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}
