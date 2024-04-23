package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

	const folderName = "output_calend"
	if err := os.MkdirAll(folderName, 0755); err != nil {
		log.Fatalf("error creating directory: %v", err)
	}

	file, err := os.Create(folderName + "/insert_holidays.sql")
	if err != nil {
		log.Fatal("Error creating SQL file:", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	_, err = writer.WriteString("-- Bulk Insert Into Holidays Table\n")
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}
	_, err = writer.WriteString("INSERT INTO holidays (order_number, date, name) VALUES\n")
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}

	firstEntry := true
	for _, month := range months {
		url := fmt.Sprintf("https://www.calend.ru/holidays/%s/", month)
		firstEntry = processMonth(url, allowedCategories, writer, firstEntry)
	}

	if !firstEntry {
		// Backtrack to overwrite the last comma and write the semicolon
		if _, err := writer.WriteString(";"); err != nil {
			log.Fatal("Error writing final semicolon:", err)
		}
	}

	writer.Flush()
}

func processMonth(url string, allowedCategories map[string]bool, writer *bufio.Writer, firstEntry bool) bool {
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
					formattedName := fmt.Sprintf("%s [%s]", holidayName, strings.Join(categories, ", "))
					sql := fmt.Sprintf("(%d, '%s', '%s')", orderNumber, date, formattedName)
					if !firstEntry {
						sql = ",\n" + sql // Prefix a comma if it's not the first entry
					}
					_, err := writer.WriteString(sql)
					if err != nil {
						log.Fatal("Error writing to file:", err)
					}
					firstEntry = false
					orderNumber++
				}
			})
		}
	})
	return firstEntry
}
