package email

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kwabena369/scrapper/internal/models"
	"gopkg.in/gomail.v2"
)

var mailClient *gomail.Dialer

func InitEmailClient() {
	host := "smtp.gmail.com"
	port := 587
	username := os.Getenv("EMAIL_USER")
	password := os.Getenv("EMAIL_PASS")

	if username == "" || password == "" {
		log.Fatal("EMAIL_USER and EMAIL_PASS must be set in environment variables")
	}

	mailClient = gomail.NewDialer(host, port, username, password)
	mailClient.TLSConfig = nil // Use STARTTLS
}

func SendFeedUpdateEmail(to, username, feedName string, newItems []models.FeedItem) error {
	if mailClient == nil {
		return fmt.Errorf("email client not initialized")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("Scrapper Team <%s>", os.Getenv("EMAIL_USER")))
	m.SetHeader("To", to)
	m.SetHeader("Subject", fmt.Sprintf("New Items in Your Followed Feed: %s", feedName))
	m.SetHeader("X-Priority", "3")
	m.SetHeader("X-MSMail-Priority", "Normal")

	// Generate HTML body
	itemsList := ""
	for i, item := range newItems {
		itemsList += fmt.Sprintf(`
			<li style="margin-bottom: 10px;">
				<a href="%s" style="color: #0066cc; text-decoration: none; font-weight: 600;">%s</a>
				<p style="color: #666666; margin: 5px 0 0 0;">%s</p>
				<p style="color: #888888; font-size: 12px; margin: 5px 0 0 0;">Published: %s</p>
			</li>`,
			item.Link,
			item.Title,
			item.Description,
			item.PubDate.Format("Jan 02, 2006"),
		)
		if i < len(newItems)-1 {
			itemsList += `<hr style="border: 0; border-top: 1px solid #eeeeee; margin: 10px 0;" />`
		}
	}

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<meta http-equiv="X-UA-Compatible" content="ie=edge">
			<title>Feed Update Notification</title>
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f4f4f9; margin: 0; padding: 0; }
				.container { max-width: 600px; margin: 20px auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1); }
				.header { background-color: #0066cc; color: #ffffff; padding: 20px; text-align: center; }
				.content { padding: 30px; color: #333333; line-height: 1.6; }
				.footer { text-align: center; padding: 20px; font-size: 12px; color: #666666; }
				@media (max-width: 600px) { .content { padding: 20px; } }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1 style="margin: 10px 0; font-size: 24px;">New Items in %s</h1>
				</div>
				<div class="content">
					<p style="font-size: 16px;">Hello %s,</p>
					<p>We found %d new item(s) in the feed <strong>%s</strong> that you follow:</p>
					<ul style="list-style: none; padding: 0;">
						%s
					</ul>
					<p style="margin-top: 20px;">Keep up with the latest updates by visiting your dashboard!</p>
				</div>
				<div class="footer">
					<p>Sent on %s</p>
					<p>Scrapper • All rights reserved</p>
				</div>
			</div>
		</body>
		</html>
	`,
		feedName,
		username,
		len(newItems),
		feedName,
		itemsList,
		time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	)

	// Generate plain text body
	itemsText := ""
	for i, item := range newItems {
		itemsText += fmt.Sprintf("%s\n%s\nPublished: %s\n%s\n", item.Title, item.Link, item.PubDate.Format("Jan 02, 2006"), item.Description)
		if i < len(newItems)-1 {
			itemsText += strings.Repeat("-", 50) + "\n"
		}
	}

	plainBody := fmt.Sprintf(
		`Hello %s,

We found %d new item(s) in the feed "%s" that you follow:

%s

Keep up with the latest updates by visiting your dashboard!

Sent on %s
Scrapper • All rights reserved`,
		username,
		len(newItems),
		feedName,
		itemsText,
		time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	)

	m.SetBody("text/plain", plainBody)
	m.AddAlternative("text/html", htmlBody)

	if err := mailClient.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email to %s: %v", to, err)
	}
	return nil
}