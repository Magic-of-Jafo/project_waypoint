package parser

import (
	"fmt"
	"log"

	// "os" // No longer needed as TestMain handles log output configuration
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ExtractAuthorUsername extracts the author's username from a post HTML block.
// postHTMLBlock is a goquery.Document created from a string starting with <tr>...</tr>.
// GoQuery wraps this in <html><body>...</body></html>, and the <td> elements
// of the original <tr> become direct children of <body>.
func ExtractAuthorUsername(postHTMLBlock *goquery.Document) (string, error) {
	const tdSelector = "td.normal.bgc1.c.w13.vat"
	authorCell := postHTMLBlock.Find(tdSelector).First()
	if authorCell.Length() == 0 {
		bodyHTML, _ := postHTMLBlock.Find("body").First().Html()
		log.Printf("[DEBUG Parser EAU] Author cell ('%s') not found. Body HTML:\n%s", tdSelector, bodyHTML)
		return "", fmt.Errorf("EAU: author cell ('%s') not found", tdSelector)
	}

	strongEl := authorCell.ChildrenFiltered("strong").First()
	if strongEl.Length() == 0 {
		acHtml, _ := authorCell.Html()
		log.Printf("[DEBUG Parser EAU] Author cell found, but no strong tag. Author cell HTML:\n%s", acHtml)
		return "", fmt.Errorf("EAU: strong element (username) not found as first strong child within author cell")
	}
	return strings.TrimSpace(strongEl.Text()), nil
}

// ExtractTimestamp extracts and parses the post timestamp.
// postHTMLBlock is a goquery.Document created from a string starting with <tr>...</tr>.
func ExtractTimestamp(postHTMLBlock *goquery.Document) (string, error) {
	const tdSelector = "td.normal.bgc1.vat.w90"
	timestampCell := postHTMLBlock.Find(tdSelector).First()
	if timestampCell.Length() == 0 {
		bodyHTML, _ := postHTMLBlock.Find("body").First().Html()
		log.Printf("[DEBUG Parser ET] Timestamp cell ('%s') not found. Body HTML:\n%s", tdSelector, bodyHTML)
		return "", fmt.Errorf("ET: timestamp cell ('%s') not found", tdSelector)
	}

	const spanBSelector = "div.vt1.liketext > div.like_left > span.b"
	spanB := timestampCell.Find(spanBSelector).First()
	if spanB.Length() == 0 {
		tcHtml, _ := timestampCell.Html()
		log.Printf("[DEBUG Parser ET] Timestamp cell found, but no span.b with selector '%s'. Timestamp cell HTML:\n%s", spanBSelector, tcHtml)
		return "", fmt.Errorf("ET: timestamp span.b not found with selector '%s' within timestamp cell", spanBSelector)
	}

	rawTimestamp := strings.TrimSpace(spanB.Text())
	rawTimestamp = strings.ReplaceAll(rawTimestamp, "\u00a0", " ")
	rawTimestamp = strings.TrimSpace(rawTimestamp)
	if strings.HasPrefix(rawTimestamp, "Posted: ") {
		rawTimestamp = strings.TrimPrefix(rawTimestamp, "Posted: ")
	}

	if rawTimestamp == "" {
		log.Printf("[DEBUG Parser ET] Timestamp string was empty after trimming. Original spanB text: '%s'", spanB.Text())
		return "", fmt.Errorf("ET: timestamp string was empty after trimming")
	}

	layouts := []string{
		"Jan _2, 2006 03:04 pm",
		"Jan _2, 2006 3:04 pm",
	}
	var t time.Time
	var parseErr error
	parsed := false
	for _, layout := range layouts {
		t, parseErr = time.Parse(layout, rawTimestamp)
		if parseErr == nil {
			parsed = true
			break
		}
	}
	if !parsed {
		log.Printf("[DEBUG Parser ET] Failed to parse timestamp '%s' with known layouts. Last error: %v", rawTimestamp, parseErr)
		return "", fmt.Errorf("ET: failed to parse timestamp '%s' with known layouts: %w", rawTimestamp, parseErr)
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

// ExtractPostID extracts the post ID from a post HTML block.
// postHTMLBlock is a goquery.Document created from a string starting with <tr>...</tr>.
func ExtractPostID(postHTMLBlock *goquery.Document) (string, error) {
	const tdSelector = "td.normal.bgc1.vat.w90"
	timestampCell := postHTMLBlock.Find(tdSelector).First()
	if timestampCell.Length() == 0 {
		bodyHTML, _ := postHTMLBlock.Find("body").First().Html()
		log.Printf("[DEBUG Parser EPI] Timestamp cell ('%s') not found (for PostID). Body HTML:\n%s", tdSelector, bodyHTML)
		return "", fmt.Errorf("EPI: timestamp cell ('%s') not found for PostID extraction", tdSelector)
	}

	const postIDSpecificSelector = "div.vt1.liketext > div.like_right > span[id^=p_]"
	postIDStr := ""
	var extractionErr error
	timestampCell.Find(postIDSpecificSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		id, exists := s.Attr("id")
		if !exists {
			extractionErr = fmt.Errorf("EPI: found post ID element but it has no id attribute")
			return false // break
		}
		if !strings.HasPrefix(id, "p_") {
			extractionErr = fmt.Errorf("EPI: found post ID '%s' but it does not start with 'p_'", id)
			return false // break
		}
		postIDStr = strings.TrimPrefix(id, "p_")
		if postIDStr == "" {
			extractionErr = fmt.Errorf("EPI: extracted post ID from attribute '%s' was empty", id)
			return false // break
		}
		return false // Stop after the first match
	})
	if extractionErr != nil {
		log.Printf("[DEBUG Parser EPI] Error during EachWithBreak for PostID: %v", extractionErr)
		return "", extractionErr
	}
	if postIDStr == "" {
		cellHtml, _ := timestampCell.Html()
		log.Printf("[DEBUG Parser EPI] Post ID element not found with selector: '%s'. Timestamp cell HTML:\n%s", postIDSpecificSelector, cellHtml)
		return "", fmt.Errorf("EPI: post ID element not found with selector: %s in timestamp cell", postIDSpecificSelector)
	}
	return postIDStr, nil
}

// ExtractPostOrderOnPage extracts the post's order on the page.
// postHTMLBlock is a goquery.Document created from a string starting with <tr>...</tr>.
func ExtractPostOrderOnPage(postHTMLBlock *goquery.Document) (int, error) {
	const tdSelector = "td.normal.bgc1.vat.w90"
	timestampCell := postHTMLBlock.Find(tdSelector).First()
	if timestampCell.Length() == 0 {
		bodyHTML, _ := postHTMLBlock.Find("body").First().Html()
		log.Printf("[DEBUG Parser EPOP] Timestamp cell ('%s') not found (for PostOrder). Body HTML:\n%s", tdSelector, bodyHTML)
		return 0, fmt.Errorf("EPOP: timestamp cell ('%s') not found for PostOrder extraction", tdSelector)
	}

	const postOrderSpecificSelector = "div.vt1.liketext > div.like_left > a[name]"
	postOrderStr := ""
	var extractionErr error
	timestampCell.Find(postOrderSpecificSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		nameAttr, exists := s.Attr("name")
		if !exists {
			extractionErr = fmt.Errorf("EPOP: found post order anchor but it has no name attribute")
			return false // break
		}
		postOrderStr = nameAttr
		if postOrderStr == "" {
			extractionErr = fmt.Errorf("EPOP: extracted post order from name attribute was empty")
			return false // break
		}
		return false // Stop after finding the first one
	})
	if extractionErr != nil {
		log.Printf("[DEBUG Parser EPOP] Error during EachWithBreak for PostOrder: %v", extractionErr)
		return 0, extractionErr
	}
	if postOrderStr == "" {
		cellHtml, _ := timestampCell.Html()
		// Make the log message unmistakable to confirm recompilation
		log.Printf("EPOP: [ACTUAL SELECTOR TEST v3] Using selector: %q. Anchor not found. Cell HTML:\n%s", postOrderSpecificSelector, cellHtml)
		return 0, fmt.Errorf("EPOP: post order anchor element not found with selector: %s in timestamp cell", postOrderSpecificSelector)
	}

	postOrder, err := strconv.Atoi(postOrderStr)
	if err != nil {
		log.Printf("[DEBUG Parser EPOP] Failed to convert post order '%s' to integer: %v", postOrderStr, err)
		return 0, fmt.Errorf("EPOP: failed to convert post order '%s' to integer: %w", postOrderStr, err)
	}
	return postOrder, nil
}
