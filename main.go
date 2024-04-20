package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

var monthTranslations = map[string]string{
	"January":   "yanvar",
	"February":  "fevral",
	"March":     "mart",
	"April":     "aprel",
	"May":       "may",
	"June":      "iyun",
	"July":      "iyul",
	"August":    "avgust",
	"September": "sentyabr",
	"October":   "oktyabr",
	"November":  "noyabr",
	"December":  "dekabr",
}

func main() {
	start := time.Now()

	baseURL, exists := os.LookupEnv("BASE_URL")
	if !exists {
		log.Fatal("environment variable BASE_URL not set")
	}

	const folderName = "output"
	if err := os.MkdirAll(folderName, 0755); err != nil {
		log.Fatalf("error creating directory: %v", err)
	}

	const currentYear = 2024
	processYear(currentYear, baseURL, folderName)

	log.Printf("Time Elapsed %s", time.Since(start))
}

func processYear(currentYear int, baseURL, folderName string) {
	for month := time.January; month <= time.December; month++ {
		for day := 1; day <= daysInMonth(month, currentYear); day++ {
			date := time.Date(currentYear, month, day, 0, 0, 0, 0, time.UTC)
			formattedDate := date.Format("2006-01-02") // Formats date as "YYYY-MM-DD"

			url := fmt.Sprintf("%s/%s/%d", baseURL, monthTranslations[month.String()], day)
			filename := fmt.Sprintf("%s/%s.html", folderName, formattedDate)

			// Skip fetch if the file already exists
			if _, err := os.Stat(filename); err == nil {
				log.Printf("Skipping fetch for %s: file already exists", filename)
				continue
			}

			start := time.Now()
			if err := fetchAndSaveHTML(url, filename); err != nil {
				log.Printf("Error fetching page for %s: %v", formattedDate, err)
				continue
			}

			log.Printf("Date: %s, File: %s, Time Elapsed for Fetch and Save: %s", date, filename, time.Since(start))
		}
	}
}

// Returns the number of days in the given month of the year
func daysInMonth(month time.Month, year int) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func fetchAndSaveHTML(url, outputFile string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithLogf(log.New(io.Discard, "", 0).Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var siteHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("html", chromedp.ByQuery),
		chromedp.OuterHTML("html", &siteHTML, chromedp.ByQuery),
	)
	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, []byte(siteHTML), 0644)
}
