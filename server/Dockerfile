FROM golang:1.25-alpine AS build

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/messenger ./cmd/messenger

FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget \
	&& adduser -D -H -s /sbin/nologin appuser

COPY --from=build /out/messenger /usr/local/bin/messenger

USER appuser

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/messenger"]
