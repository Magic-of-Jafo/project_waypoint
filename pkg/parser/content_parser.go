package parser

import (
	"fmt"
	"log"
	"project-waypoint/pkg/data"
	"regexp"
	"strings"

	// "golang.org/x/net/html" // No longer needed as extractText is commented out

	"github.com/PuerkitoBio/goquery"
)

var (
	// Handles [b]text[/b], [i]text[/i], [u]text[/u]
	// Group 1: tag name (b, i, u)
	// Group 2: content
	safeBasicBBCodeRegex = regexp.MustCompile(`(?is)\\[(b|i|u)\\](.*?)\\[/\\1\\]`)
)

// ParseContentBlocks takes a goquery selection representing the direct children
// of a post's content area and parses it into an ordered list of ContentBlock structs.
// It identifies sequences of the author's new_text and distinct quote blocks.
func ParseContentBlocks(contentNodes *goquery.Selection) ([]data.ContentBlock, error) {
	var blocks []data.ContentBlock
	var currentNewText string

	// The incoming 'contentNodes' is assumed to be the selection
	// of the actual content container (e.g., a 'div.w100').

	if contentNodes.Length() == 0 {
		// This case might occur if the parent find operation yielded nothing
		// and an empty selection is passed.
		log.Printf("Warning: ParseContentBlocks received an empty or non-matching selection for content parsing.")
		return blocks, nil // Or an error depending on desired strictness
	}

	contentNodes.Contents().Each(func(i int, s *goquery.Selection) {
		// Check if the node is a quote table
		if s.Is("table.cfq") {
			// Flush any pending new_text
			trimmedNewText := strings.TrimSpace(currentNewText)
			if len(trimmedNewText) > 0 {
				blocks = append(blocks, data.ContentBlock{Type: data.ContentBlockTypeNewText, Content: trimmedNewText})
			}
			currentNewText = ""

			quotedUser, quotedTimestamp, quotedText, err := ExtractQuoteDetails(s)
			if err != nil {
				log.Printf("Error extracting quote details: %v. Post ID or other identifier would be useful here.", err)
				// AC10: Log error and continue. Add a block indicating error.
				blocks = append(blocks, data.ContentBlock{Type: data.ContentBlockTypeQuote, QuotedUser: "ERROR_PARSING_DETAILS", QuotedText: fmt.Sprintf("Error during quote parsing: %v", err)})
			} else {
				blocks = append(blocks, data.ContentBlock{
					Type:            data.ContentBlockTypeQuote,
					QuotedUser:      quotedUser,
					QuotedTimestamp: quotedTimestamp,
					QuotedText:      quotedText,
				})
			}
		} else {
			// Node is part of new_text. Get its HTML content.
			htmlContent, err := goquery.OuterHtml(s)
			if err == nil {
				currentNewText += htmlContent
			}
			// Alternative: Get text content: currentNewText += s.Text()
			// Story 3.3 implies raw extracted text, Story 3.4 handles cleaning.
			// Using OuterHtml to preserve links, italics, etc. for now.
		}
	})

	// Flush any remaining new_text after the loop
	trimmedNewText := strings.TrimSpace(currentNewText)
	if len(trimmedNewText) > 0 {
		blocks = append(blocks, data.ContentBlock{Type: data.ContentBlockTypeNewText, Content: trimmedNewText})
	}

	return blocks, nil
}

// ExtractQuoteDetails parses a quote HTML element (expected to be a table.cfq)
// and extracts the quoted user, timestamp (if available), and the quote text.
func ExtractQuoteDetails(quoteElement *goquery.Selection) (quotedUser string, quotedTimestamp string, quotedText string, err error) {
	// Selector for the attribution cell (contains user and timestamp)
	// Typically the first <td> within the table.cfq that has a <b> tag for the username.
	attributionCell := quoteElement.Find("td:has(b)").First()
	if attributionCell.Length() == 0 {
		// Fallback or log error: No clear attribution cell found
		// This could happen if the quote structure is different than expected.
		// Per AC10, errors should be logged by the caller.
		return "", "", "", fmt.Errorf("could not find attribution cell in quote element")
	}

	// Extract Quoted User (Subtask 3.2 & 3.5)
	userSelection := attributionCell.Find("b").First()
	rawUserText := strings.TrimSpace(userSelection.Text())

	// Handle variations like "Username wrote:" or "Quote: Username"
	if strings.HasSuffix(rawUserText, " wrote:") {
		quotedUser = strings.TrimSuffix(rawUserText, " wrote:")
	} else if strings.HasPrefix(rawUserText, "Quote: ") {
		quotedUser = strings.TrimPrefix(rawUserText, "Quote: ")
	} else {
		quotedUser = rawUserText // Assume it's just the username
	}
	quotedUser = strings.TrimSpace(quotedUser)

	// Extract Quoted Timestamp (Subtask 3.3 & 3.5)
	// Timestamp is often in the same cell, sometimes after a <br> or as part of the text.
	// We'll take the full text of the attribution cell and try to parse out the timestamp.
	plainAttributionText := strings.TrimSpace(attributionCell.Text())

	// Regex to find a pattern like "On Jan 23, 2003, 07:22 AM" or just the date/time
	// This regex is a starting point and might need refinement for various forum date formats.
	// Example formats: "Jan 23, 2003, 07:22 AM", "Today at 02:10 PM", "Yesterday at 02:10 PM"
	// AC5: "parse into a consistent format (quoted_timestamp). If not found, this field should be null/empty."
	// AC7: "handle variations in quote attribution formats"

	// More complex regex might be needed. For now, a simple placeholder for structure.
	// This part is highly dependent on the actual timestamp formats encountered.
	// Example: trying to find something like Month Day, Year, HH:MM AM/PM
	reTimestamp := regexp.MustCompile(`(?i)(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},\s+\d{4},\s+\d{1,2}:\d{2}\s+(AM|PM)`)
	match := reTimestamp.FindStringSubmatch(plainAttributionText)
	if len(match) > 0 {
		rawTimestamp := match[0]
		// TODO: Parse rawTimestamp into "YYYY-MM-DD HH:MM:SS" format.
		// This requires time.Parse with the correct layout string.
		// Handling "Today", "Yesterday" would require a reference date (e.g., post's main timestamp or processing date).
		// For now, we'll store the extracted raw timestamp if found.
		quotedTimestamp = rawTimestamp // Placeholder - actual parsing needed
	} else {
		quotedTimestamp = "" // Or null, as per AC5
	}

	// Extract Quoted Text (Subtask 3.4)
	// The quoted text is usually in the next <td> sibling to the attributionCell\'s parent <tr>, or a td not being the attribution cell.
	// Simpler: find all <td>s in the quoteElement, the one that is not attributionCell is the text cell.
	quoteElement.Find("td").Each(func(i int, td *goquery.Selection) {
		if len(td.Nodes) > 0 && len(attributionCell.Nodes) > 0 &&
			td.Nodes[0] != attributionCell.Nodes[0] &&
			td.Parent().Nodes[0] == attributionCell.Parent().Nodes[0] {

			if !strings.Contains(td.Text(), quotedUser) {
				html, err := td.Html()
				if err == nil {
					quotedText = strings.TrimSpace(html)
				}
			}
		}
	})
	if quotedText == "" {
		quoteElement.Find("td").FilterFunction(func(i int, s *goquery.Selection) bool {
			return len(s.Nodes) > 0 && len(attributionCell.Nodes) > 0 && s.Nodes[0] != attributionCell.Nodes[0]
		}).EachWithBreak(func(i int, td *goquery.Selection) bool {
			if td.ParentsFiltered("table.cfq").Length() == 1 {
				html, err := td.Html()
				if err == nil {
					quotedText = strings.TrimSpace(html)
					return false // Break the loop
				}
			}
			return true
		})
	}

	// If quotedUser is empty and we have text, it might be a simple blockquote without clear attribution
	if quotedUser == "" && quotedText == "" && attributionCell.Length() == 0 {
		// Could be a simple <blockquote> or similar not matching table.cfq structure
		// For now, if we reached here with `quoteElement` being `table.cfq`, this case is less likely.
		// If quoteElement could be other things, this might be a plain quote.
		text, _ := quoteElement.Html()
		return "", "", strings.TrimSpace(text), nil // Treat whole content as text if no user
	}

	return strings.TrimSpace(quotedUser), strings.TrimSpace(quotedTimestamp), quotedText, nil
}

// processBBCodes removes common BBCode tags from a string and logs actions.
// CURRENTLY SIMPLIFIED to only handle [b], [i], [u] to avoid regex panics.
func processBBCodes(text string) string {
	cleanedText := safeBasicBBCodeRegex.ReplaceAllStringFunc(text, func(match string) string {
		submatches := safeBasicBBCodeRegex.FindStringSubmatch(match)
		if len(submatches) == 3 { // 0: full match, 1: tag, 2: content
			tag := submatches[1]
			content := submatches[2]
			log.Printf("INFO: BBCode: Simplified removal of tag [%s], preserving content: '%s'", tag, content)
			return content
		}
		return match // Should not happen with this regex if it matches
	})

	// TODO: Add back other BBCode handling (url, img, list, quote, etc.) carefully
	// For now, just return the text after [b], [i], [u] are stripped.

	return cleanedText
}

// CleanNewTextBlock converts specific HTML entities to their character equivalents,
// removes BBCode, and trims whitespace from a block of text presumed to be new content.
func CleanNewTextBlock(text string) (string, error) {
	// AC4.1: Decode HTML entities
	decodedText := strings.ReplaceAll(text, "&nbsp;", " ")
	decodedText = strings.ReplaceAll(decodedText, "&amp;", "&")
	decodedText = strings.ReplaceAll(decodedText, "&lt;", "<")
	decodedText = strings.ReplaceAll(decodedText, "&gt;", ">")
	decodedText = strings.ReplaceAll(decodedText, "&quot;", "\"")
	// Add more entities as needed, e.g., &apos; for ''

	// AC4.3: Remove BBCode (using the simplified version for now)
	bbCodeFreeText := processBBCodes(decodedText)

	// AC4.2 & AC4.4: Trim leading/trailing whitespace and ensure no excessive internal spacing
	// strings.TrimSpace handles leading/trailing. Collapsing internal spaces is more complex.
	// For now, simple trim. A more robust solution might involve splitting by space, filtering empty strings, and rejoining.
	finalText := strings.TrimSpace(bbCodeFreeText)

	// Story 5.1.5 AC5: If, after all cleaning, the block is empty, it should be discarded.
	// This function returns the cleaned string; the decision to discard is up to the caller.
	return finalText, nil
}

/*
// extractText recursively traverses HTML nodes and appends text content to the strings.Builder.
func extractText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	} else if n.Type == html.ElementNode && n.Data == "img" {
		var altText, titleText string
		imgSrc := ""
		for _, attr := range n.Attr {
			if attr.Key == "alt" {
				altText = attr.Val
			}
			if attr.Key == "title" {
				titleText = attr.Val
			}
			if attr.Key == "src" {
				imgSrc = attr.Val
			}
		}

		if altText != "" {
			sb.WriteString(altText)
			log.Printf("INFO: Replaced <img> tag (src: %s) with alt text: '%s'", imgSrc, altText) // AC9
		} else if titleText != "" {
			sb.WriteString(titleText)
			log.Printf("INFO: Replaced <img> tag (src: %s) with title text: '%s'", imgSrc, titleText) // AC9
		} else {
			sb.WriteString("[image]")                                                                                         // AC2.4 Placeholder
			log.Printf("INFO: Replaced <img> tag (src: %s) with placeholder '[image]' as no alt or title text found", imgSrc) // AC9
		}
		// Do not traverse children of <img> tags as they are void elements
		return
	} else if n.Type == html.ElementNode && n.Data == "br" {
		sb.WriteString("\n") // AC3.1
		// Do not traverse children of <br> tags as they are void elements
		return
	}

	// Child traversal
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			// Do not traverse into script or style tags
			continue
		}
		extractText(c, sb)
	}

	// Add a newline after certain block-level elements for readability (AC3.2)
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "blockquote", "figure", "hr", "table", "ul", "ol", "dl", "section", "article", "header", "footer", "aside", "nav":
			// Add newline if not already ending with one or more newlines
			// or multiple spaces which will be collapsed later.
			// This helps ensure separation after block elements are processed.
			// The final whitespace normalization step will clean up extra newlines.
			sb.WriteString("\n")
		}
	}
}
*/
