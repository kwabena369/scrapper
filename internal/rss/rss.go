package rss

import (
	"encoding/xml"
	"log"
	"net/http"
)

// RSS represents the structure of an RSS feed
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel holds the feed metadata and items
type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

// Item represents a single entry in the RSS feed
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// FetchRSS fetches and parses an RSS feed from a given URL
func FetchRSS(url string) ([]Item, error) {
	// Make HTTP request to fetch the feed
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch RSS feed from %s: %s", url, resp.Status)
		return nil, err
	}

	// Parse the XML response
	var rss RSS
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&rss)
	if err != nil {
		log.Printf("Failed to parse RSS feed from %s: %v", url, err)
		return nil, err
	}

	return rss.Channel.Items, nil
}