package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// SubForum holds the extracted information for a sub-forum
type SubForum struct {
	ID                    string
	Name                  string
	BaseURL               string
	Description           string
	TopicsCount           string
	PostsCount            string
	LastActiveDateTimeStr string
	LastActiveBy          string
	LastPostID            string
}

func main() {
	outputDirFlag := flag.String("outputDir", "data", "Directory to save the output CSV file")
	outputFileFlag := flag.String("outputFile", "subforum_list.csv", "Name of the output CSV file")
	flag.Parse()

	inputFileValue := "bmad-agent\\forum_front_page.html" // Hardcoded path

	file, err := os.Open(inputFileValue)
	if err != nil {
		log.Fatalf("Error opening input file %s: %v", inputFileValue, err)
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(bufio.NewReader(file))
	if err != nil {
		log.Fatalf("Error parsing HTML from file %s: %v", inputFileValue, err)
	}

	var subForums []SubForum

	doc.Find("table.normal").Each(func(i int, tableSel *goquery.Selection) {
		// Iterate over rows that represent forums (tr elements with td.bgc2)
		tableSel.Find("tr").Each(func(j int, rowSel *goquery.Selection) {
			if rowSel.Find("td.bgc2 a.b[href*='viewforum.php']").Length() > 0 {
				// This row looks like a sub-forum entry
				sf := SubForum{}

				// Extract Name and BaseURL
				linkSel := rowSel.Find("td.bgc2 a.b[href*='viewforum.php']")
				sf.Name = strings.TrimSpace(linkSel.Text())
				baseURL, _ := linkSel.Attr("href")
				if baseURL != "" {
					// Make URL absolute if it's relative
					if !strings.HasPrefix(baseURL, "http") {
						sf.BaseURL = "https://www.themagiccafe.com/forums/" + baseURL
					} else {
						sf.BaseURL = baseURL
					}
					// Extract ID from BaseURL
					u, err := url.Parse(sf.BaseURL)
					if err == nil {
						q := u.Query()
						sf.ID = q.Get("forum")
					}
				}

				// Extract Description
				descriptionSel := linkSel.Parent().Find("span.smalltext")
				sf.Description = strings.TrimSpace(descriptionSel.First().Text()) // First() in case of multiple smalltext spans

				// Extract Topics and Posts counts
				countsSel := rowSel.Find("td.bgc2.w5.c.normal.midtext")
				if countsSel.Length() >= 2 {
					sf.TopicsCount = strings.TrimSpace(countsSel.Eq(0).Text())
					sf.PostsCount = strings.TrimSpace(countsSel.Eq(1).Text())
				}

				// Skip "Chef's Specials By Year:" pseudo-forum entry
				if strings.Contains(sf.Name, "Chef's Specials By Year") {
					return // skip this iteration
				}
				// Skip "Welcome special guest of honor" if it has 0 topics/posts, as it's often a placeholder
				if strings.Contains(sf.Name, "Welcome special guest of honor") && sf.TopicsCount == "0" && sf.PostsCount == "0" {
					return
				}

				// Extract Last Active Info
				lastActiveCell := rowSel.Find("td.bgc2.w22.c.normal")
				if lastActiveCell.Length() > 0 {
					lastActiveDateTimeText := lastActiveCell.Find("span.midtext").First().Text()
					sf.LastActiveDateTimeStr = strings.TrimSpace(lastActiveDateTimeText)

					if sf.LastActiveDateTimeStr != "No Posts" {
						lastActiveByText := lastActiveCell.Find("span.smalltext").First().Contents().Not("a").Text()
						sf.LastActiveBy = strings.TrimSpace(strings.Replace(lastActiveByText, "by ", "", 1))

						lastPostLinkSel := lastActiveCell.Find("span.smalltext a.b[href*='viewtopic.php']")
						if lastPostLinkSel.Length() > 0 {
							lastPostURL, _ := lastPostLinkSel.Attr("href")
							u, err := url.Parse(lastPostURL)
							if err == nil {
								q := u.Query()
								sf.LastPostID = q.Get("post")
								if sf.LastPostID == "" { // sometimes it's in topic=123&post=456 format
									sf.LastPostID = q.Get("topic") // fallback, less ideal
								}
							}
						}
					}
				}
				// Only add if we have an ID and Name, to avoid partial entries or headers
				if sf.ID != "" && sf.Name != "" {
					// Basic filter for some known non-forum rows that might have viewforum.php links
					// but are not actual subforums, e.g. category headers themselves if they have such a link by mistake
					if !strings.Contains(sf.Name, "Mark all forums as read") && !strings.Contains(sf.BaseURL, "viewcat=") {
						subForums = append(subForums, sf)
					}
				}
			}
		})
	})

	// Remove duplicates based on SubForum.ID
	uniqueSubForumsMap := make(map[string]SubForum)
	for _, sf := range subForums {
		if _, exists := uniqueSubForumsMap[sf.ID]; !exists {
			if sf.ID != "" { // Ensure we only add entries with an ID
				uniqueSubForumsMap[sf.ID] = sf
			}
		}
	}

	var uniqueSubForums []SubForum
	for _, sf := range uniqueSubForumsMap {
		uniqueSubForums = append(uniqueSubForums, sf)
	}

	outputDirValue := *outputDirFlag
	outputFileValue := *outputFileFlag

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDirValue, os.ModePerm); err != nil {
		log.Fatalf("Error creating output directory %s: %v", outputDirValue, err)
	}

	// Write to CSV
	outputPath := filepath.Join(outputDirValue, outputFileValue)
	csvFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Error creating CSV file %s: %v", outputPath, err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	headers := []string{
		"sub_forum_id", "sub_forum_name", "base_url", "description",
		"topics_count", "posts_count", "last_active_datetime_str",
		"last_active_by", "last_post_id",
	}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Error writing CSV headers: %v", err)
	}

	for _, sf := range uniqueSubForums {
		row := []string{
			sf.ID, sf.Name, sf.BaseURL, sf.Description,
			sf.TopicsCount, sf.PostsCount, sf.LastActiveDateTimeStr,
			sf.LastActiveBy, sf.LastPostID,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Error writing row to CSV for sub-forum ID %s: %v. Skipping row.", sf.ID, err)
		}
	}

	log.Printf("Successfully parsed %d unique sub-forums and wrote to %s", len(uniqueSubForums), outputPath)
}

// Helper to extract forum ID, more robustly if needed
var forumIDRegex = regexp.MustCompile(`forum=(\d+)`)

func extractForumID(urlStr string) string {
	matches := forumIDRegex.FindStringSubmatch(urlStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
