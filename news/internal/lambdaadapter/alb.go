// Package lambdaadapter serves a net/http handler from an ALB Lambda target.
package lambdaadapter

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Handle converts an ALB target group request into an http.Request, runs h and
// converts the recorded response back.
func Handle(h http.Handler) func(context.Context, events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	return func(ctx context.Context, req events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
		body := req.Body
		if req.IsBase64Encoded {
			dec, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				return events.ALBTargetGroupResponse{StatusCode: http.StatusBadRequest, StatusDescription: "400 Bad Request"}, nil
			}
			body = string(dec)
		}

		query := url.Values{}
		for k, v := range req.MultiValueQueryStringParameters {
			for _, s := range v {
				query.Add(k, decode(s))
			}
		}
		for k, v := range req.QueryStringParameters {
			if _, ok := req.MultiValueQueryStringParameters[k]; !ok {
				query.Add(k, decode(v))
			}
		}

		r, err := http.NewRequestWithContext(ctx, req.HTTPMethod,
			(&url.URL{Path: req.Path, RawQuery: query.Encode()}).String(), strings.NewReader(body))
		if err != nil {
			return events.ALBTargetGroupResponse{StatusCode: http.StatusBadRequest, StatusDescription: "400 Bad Request"}, nil
		}
		for k, v := range req.MultiValueHeaders {
			for _, s := range v {
				r.Header.Add(k, s)
			}
		}
		for k, v := range req.Headers {
			if _, ok := req.MultiValueHeaders[k]; !ok {
				r.Header.Set(k, v)
			}
		}
		r.Host = r.Header.Get("Host")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		res := rec.Result()
		defer res.Body.Close()

		out := events.ALBTargetGroupResponse{
			StatusCode:        res.StatusCode,
			StatusDescription: http.StatusText(res.StatusCode),
			Headers:           map[string]string{},
			Body:              rec.Body.String(),
		}
		for k := range res.Header {
			out.Headers[k] = res.Header.Get(k)
		}
		return out, nil
	}
}

// ALB passes query values URL-encoded; undo that once before re-encoding.
func decode(s string) string {
	if d, err := url.QueryUnescape(s); err == nil {
		return d
	}
	return s
}
