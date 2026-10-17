package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type fakeCW struct {
	in  *cloudwatch.PutMetricDataInput
	err error
}

func (f *fakeCW) PutMetricData(_ context.Context, in *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	f.in = in
	return &cloudwatch.PutMetricDataOutput{}, f.err
}

func TestCollectPublishesThreeCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`FROM messages.*FROM chats.*FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"messages", "chats", "users"}).AddRow(10, 3, 5))

	cw := &fakeCW{}
	if err := Collect(context.Background(), db, cw, "E2EEText/Messenger", "dev", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	if *cw.in.Namespace != "E2EEText/Messenger" || len(cw.in.MetricData) != 3 {
		t.Fatalf("input = %+v", cw.in)
	}
	want := map[string]float64{"Messages": 10, "Chats": 3, "Users": 5}
	for _, d := range cw.in.MetricData {
		if want[*d.MetricName] != *d.Value {
			t.Errorf("%s = %v, want %v", *d.MetricName, *d.Value, want[*d.MetricName])
		}
		if len(d.Dimensions) != 1 || *d.Dimensions[0].Name != "Environment" || *d.Dimensions[0].Value != "dev" {
			t.Errorf("%s dimensions = %+v", *d.MetricName, d.Dimensions)
		}
	}
}

func TestCollectOmitsEmptyEnvironment(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"m", "c", "u"}).AddRow(0, 0, 0))
	cw := &fakeCW{}
	if err := Collect(context.Background(), db, cw, "ns", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(cw.in.MetricData[0].Dimensions) != 0 {
		t.Fatalf("dimensions = %+v", cw.in.MetricData[0].Dimensions)
	}
}

func TestCollectQueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("boom"))
	cw := &fakeCW{}
	if err := Collect(context.Background(), db, cw, "ns", "dev", time.Now()); err == nil {
		t.Fatal("expected error")
	}
	if cw.in != nil {
		t.Fatal("must not publish partial data on failure")
	}
}

func TestCollectPutError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"m", "c", "u"}).AddRow(1, 1, 1))
	if err := Collect(context.Background(), db, &fakeCW{err: errors.New("denied")}, "ns", "dev", time.Now()); err == nil {
		t.Fatal("expected error")
	}
}
