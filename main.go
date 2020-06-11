package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

var reTitle = regexp.MustCompile("h2 data-attribute=\"(.+?)\"")
var reLink = regexp.MustCompile("href=\"(https[^>]+?)\"><h2 data-attribute")
var reAsin = regexp.MustCompile("dp/(.+?)/")

func main() {
	url := "https://www.amazon.co.jp/s/?node=5347026051"
	html := crawl(url)

	if html != "" {
		parse(html)
	}
}

func crawl(url string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(bytes)
}

type book struct {
	Asin      string `dynamo:"asin"`
	YearMonth string `dynamo:"year_month"`
	Date      string `dynamo:"date"`
	Title     string `dynamo:"title"`
	Categoy   int    `dynamo:"category"`
	CreatedAt string `dynamo:"created_at"`
}

var db *dynamo.DB
var table dynamo.Table

func parse(html string) {
	titles := reTitle.FindAllStringSubmatch(html, -1)

	for i, title := range titles {
		fmt.Printf("title %d: %s\n", i, title[1])
	}

	links := reLink.FindAllStringSubmatch(html, -1)

	for i, link := range links {
		fmt.Printf("link %d: %s\n", i, link[1])

		asin := reAsin.FindAllStringSubmatch(link[1], -1)
		fmt.Printf("asin %d: %s\n", i, asin[0][1])

		title := titles[i][1]
		create(title, asin[0][1])
	}
}

func create(title string, asin string) {
	now := time.Now()
	yearMonth := now.Format("2006-01")
	date := now.Format("2006-01-02")
	createdAt := now.Format(time.RFC3339)

	record := book{
		Asin:      asin,
		YearMonth: yearMonth,
		Date:      date,
		Title:     title,
		Categoy:   0,
		CreatedAt: createdAt,
	}

	err := table.Put(record).Run()

	if err == nil {
		fmt.Printf("put book: %#v\n", record)
	} else {
		fmt.Printf("Failed to put item, err: %v\n", err)
		fmt.Printf("book: %#v\n", record)
	}
}

func init() {
	dynamoDbLocal := os.Getenv("DYNAMO_DB_LOCAL")

	if dynamoDbLocal == "true" {
		db = dynamo.New(session.New(), &aws.Config{
			Region:   aws.String("ap-northeast-1"),
			Endpoint: aws.String("http://localhost:8000"),
		})
	} else {
		dynamo.New(session.New(), &aws.Config{Region: aws.String("ap-northeast-1")})
	}
	table = db.Table("prime_books")
}
