package lambdaadapter

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandle(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(r.Method + " " + r.URL.Path + " " + r.URL.Query().Get("before") + "|" + r.Header.Get("X-A")))
	})
	res, err := Handle(h)(context.Background(), events.ALBTargetGroupRequest{
		HTTPMethod:            "GET",
		Path:                  "/api/v1/news",
		QueryStringParameters: map[string]string{"before": "a%2Db"},
		Headers:               map[string]string{"x-a": "1", "host": "example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusTeapot || res.Body != "GET /api/v1/news a-b|1" {
		t.Errorf("res = %+v", res)
	}
	if res.Headers["Cache-Control"] != "public, max-age=3600" {
		t.Errorf("headers = %v", res.Headers)
	}
}
