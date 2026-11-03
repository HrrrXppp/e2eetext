// Command news-lambda serves the news API as an AWS Lambda behind an ALB target group.
package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/ekhrunov/messenger/news/internal/database"
	"github.com/ekhrunov/messenger/news/internal/lambdaadapter"
	"github.com/ekhrunov/messenger/news/internal/news"
)

// Version is set at link time.
var Version = "dev"

func main() {
	url := os.Getenv("NEWS_DATABASE_URL")
	if url == "" {
		log.Fatal("NEWS_DATABASE_URL is required")
	}
	// Pool of 1: each Lambda instance handles one request at a time.
	db, err := database.Open(url, 1)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	h := news.NewHandler(news.NewRepository(db), Version)
	lambda.Start(lambdaadapter.Handle(h))
}
