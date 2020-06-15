package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/guregu/dynamo"
)

var reMain = regexp.MustCompile("s-main-slot")
var reAsin = regexp.MustCompile("dp/(.+?)/")

var reTitleFirst = regexp.MustCompile("h2 data-attribute=\"(.+?)\"")
var reLinkFirst = regexp.MustCompile("href=\"(https[^>]+?)\"><h2 data-attribute")

var reTitle = regexp.MustCompile("(?s)s-result-item.+?s-asin.+?h2.+?a.+?span.+?>(.+?)</span>")
var reLink = regexp.MustCompile("(?s)s-result-item.+?s-asin.+?h2.+?a.+?href=\"(.+?)\"")

var client *twitter.Client

func main() {
	for i := 1; i <= 3; i++ {
		url := "https://www.amazon.co.jp/s/?node=5347026051&page=" + strconv.Itoa(i)

		fmt.Printf("url: %s\n", url)

		html := crawl(url)

		if html != "" {
			parse(html, i)
		} else {
			fmt.Println("html is empty")
		}
	}
}

func crawl(url string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("req, err: %v\n", err)
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("resp, err: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("bytes, err: %v\n", err)
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

func parse(html string, page int) {
	secondPage := reMain.MatchString(html)

	if page == 1 && secondPage {
		fmt.Println("crawlOtherPage")
		crawlOtherPage()
	} else if secondPage {
		fmt.Println("parseSecondPage")
		parseSecondPage(html)
	} else {
		fmt.Println("parseFirstPage")
		parseFirstPage(html)
	}
}

func crawlOtherPage() {
	url := "https://www.amazon.co.jp/b?node=5347026051"
	html := crawl(url)

	if html != "" {
		parse(html, 0)
	} else {
		fmt.Println("html is empty")
	}
}

func parseFirstPage(html string) {
	titles := reTitleFirst.FindAllStringSubmatch(html, -1)
	links := reLinkFirst.FindAllStringSubmatch(html, -1)

	for i, link := range links {
		asin := reAsin.FindAllStringSubmatch(link[1], -1)
		title := titles[i][1]

		tweetOnce(title, asin[0][1])

		time.Sleep(1 * time.Second)
	}

}

func parseSecondPage(html string) {
	titles := reTitle.FindAllStringSubmatch(html, -1)
	links := reLink.FindAllStringSubmatch(html, -1)

	for i, link := range links {
		asin := reAsin.FindAllStringSubmatch(link[1], -1)
		title := titles[i][1]

		tweetOnce(title, asin[0][1])

		time.Sleep(1 * time.Second)
	}
}

func tweetOnce(title string, asin string) {
	var book book
	query := table.Get("asin", asin)
	err := query.One(&book)

	if err != nil {
		err := tweet(title, asin)

		if err != nil {
			fmt.Printf("tweet, err: %v\n", err)
		} else {
			create(title, asin)
		}
	} else {
		fmt.Printf("tweeted %s\n", asin)
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
		fmt.Printf("asin: %s\n", asin)
	}
}

func tweet(title string, asin string) error {
	url := "https://www.amazon.co.jp/exec/obidos/ASIN/" + asin + "/twiaso-22/"
	text := fmt.Sprintf("%s\n%s\n#kindle #amazon", title, url)

	fmt.Printf("tweet : %s\n", asin)
	_, _, err := client.Statuses.Update(text, nil)

	return err
}

func dumpPage(html string) {
	file, err := os.Create("page.html")
	if err != nil {
		fmt.Printf("file, err: %v\n", err)
	}
	defer file.Close()

	file.Write(([]byte)(html))
}

func parseFirstPageDebug(html string) {
	titles := reTitleFirst.FindAllStringSubmatch(html, -1)
	if len(titles) == 0 {
		log.Fatal("titles is zero")
	}

	for i, title := range titles {
		fmt.Printf("title %d: %s\n", i, title[1])
	}

	links := reLinkFirst.FindAllStringSubmatch(html, -1)

	for i, link := range links {
		fmt.Printf("link %d: %s\n", i, link[1])

		asin := reAsin.FindAllStringSubmatch(link[1], -1)
		fmt.Printf("asin %d: %s\n", i, asin[0][1])
	}
}

func parseSecondPageDebug(html string) {
	titles := reTitle.FindAllStringSubmatch(html, -1)
	if len(titles) == 0 {
		log.Fatal("titles is zero")
	}

	for i, title := range titles {
		fmt.Printf("title %d: %s\n", i, title[1])
	}

	links := reLink.FindAllStringSubmatch(html, -1)

	for i, link := range links {
		fmt.Printf("link %d: %s\n", i, link[1])

		asin := reAsin.FindAllStringSubmatch(link[1], -1)
		fmt.Printf("asin %d: %s\n", i, asin[0][1])
	}
}

func init() {
	// dynamodb
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

	// Twitter client
	consumerKey := os.Getenv("API_KEY")
	consumerSecret := os.Getenv("API_SECRET")
	accessToken := os.Getenv("ACCESS_TOKEN")
	accessSecret := os.Getenv("ACCESS_TOKEN_SECRET")

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client = twitter.NewClient(httpClient)
}
