# Lambda container image (ALB target). Build context is the repo root.
FROM golang:1.25-alpine AS build

WORKDIR /src

COPY news/go.mod news/go.sum ./
RUN go mod download

COPY news/ .
COPY VERSION ./VERSION

RUN VERSION=$(cat VERSION) && \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
		-ldflags="-s -w -X main.Version=${VERSION}" -o /out/bootstrap ./cmd/news-lambda && \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
		-ldflags="-s -w -X main.Version=${VERSION}" -o /out/news ./cmd/news

FROM public.ecr.aws/lambda/provided:al2023

# Lambda runs the bootstrap; the same image runs "news migrate" as a one-off deploy step
# by overriding the entrypoint/command.
COPY --from=build /out/bootstrap /var/runtime/bootstrap
COPY --from=build /out/news /usr/local/bin/news

CMD ["bootstrap"]
