package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	months := []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}
	allowedCategories := map[string]bool{
		"Международные праздники": true,
		"Праздники России":        true,
		"Праздники славян":        true,
		"Праздники ООН":           true,
		"Православные праздники":  true,
	}

	const (
		folderName   = "output_calend"
		baseFileName = "insert_holidays_and_links.sql"
	)
	if err := os.MkdirAll(folderName, 0755); err != nil {
		log.Fatal("Error creating directory:", err)
	}

	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d_%s", timestamp, baseFileName)
	filePath := filepath.Join(folderName, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal("Error creating SQL file:", err)
	}
	defer file.Close()

	var holidayInserts []string
	var linkInserts []string

	for _, month := range months {
		url := fmt.Sprintf("https://www.calend.ru/holidays/%s/", month)
		processMonth(url, allowedCategories, &holidayInserts, &linkInserts)
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	writer.WriteString(`BEGIN;

CREATE TABLE IF NOT EXISTS holiday_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS holiday_category_links (
    holiday_id INT REFERENCES holidays(id) ON DELETE CASCADE,
    category_id INT REFERENCES holiday_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (holiday_id, category_id)
);

DELETE FROM holidays;

`)

	if len(holidayInserts) > 0 {
		writer.WriteString("INSERT INTO holidays (order_number, date, name) VALUES\n")
		writer.WriteString(strings.Join(holidayInserts, ",\n"))
		writer.WriteString(";\n")
	}

	for category := range allowedCategories {
		categoryInsert := fmt.Sprintf(`
INSERT INTO holiday_categories (name)
VALUES ('%s')
ON CONFLICT (name) DO NOTHING;`, strings.ReplaceAll(category, "'", "''"))
		writer.WriteString(categoryInsert)
		writer.WriteString("\n")
	}

	if len(linkInserts) > 0 {
		writer.WriteString(strings.Join(linkInserts, "\n"))
		writer.WriteString("\n")
	}

	writer.WriteString("COMMIT;\n")
	writer.Flush()
}

func processMonth(url string, allowedCategories map[string]bool, holidayInserts *[]string, linkInserts *[]string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Error making HTTP request:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Received non-200 response code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
	}

	var currentDay string
	orderNumber := 1

	holidays := doc.Find(".block .datesList .holidayweek ul.itemsNet li")
	holidays.Each(func(i int, item *goquery.Selection) {
		link, exists := item.Find(".dataNum a").Attr("href")
		if exists {
			date := strings.TrimPrefix(link, "/day/")
			date = strings.TrimSuffix(date, "/")

			if date != currentDay {
				orderNumber = 1
				currentDay = date
			}

			item.Find(".caption").Each(func(j int, cap *goquery.Selection) {
				holidayName := cap.Find(".title a").Text()
				holidayName = strings.ReplaceAll(holidayName, "'", "''")

				var categories []string
				cap.Find("img").Each(func(k int, img *goquery.Selection) {
					if alt, exists := img.Attr("alt"); exists && allowedCategories[alt] {
						categories = append(categories, alt)
					}
				})

				if len(categories) > 0 {
					holidayInsert := fmt.Sprintf("(%d, '%s', '%s')", orderNumber, date, holidayName)
					*holidayInserts = append(*holidayInserts, holidayInsert)

					for _, category := range categories {
						linkInsert := fmt.Sprintf(`
INSERT INTO holiday_category_links (holiday_id, category_id)
VALUES ((SELECT id FROM holidays WHERE order_number = %d AND name = '%s'), 
       (SELECT id FROM holiday_categories WHERE name = '%s'))
ON CONFLICT DO NOTHING;`, orderNumber, strings.ReplaceAll(holidayName, "'", "''"), strings.ReplaceAll(category, "'", "''"))

						*linkInserts = append(*linkInserts, linkInsert)
					}

					orderNumber++
				}
			})
		}
	})
}
