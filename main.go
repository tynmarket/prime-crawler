package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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
	}
}
