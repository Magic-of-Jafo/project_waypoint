package htmlparser

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/PuerkitoBio/goquery"
)

// HTMLPage represents a loaded HTML page from the archive
type HTMLPage struct {
	FilePath string
	Content  *goquery.Document
}

// PostBlock represents an individual post identified on a page.
// It holds the goquery.Selection for the block, allowing further specific data extraction.
// This aligns with AC6: "provide access to the isolated HTML content for each identified post block"
type PostBlock struct {
	Selection *goquery.Selection
	// We can add fields here later for extracted data like User, PostTimestamp, PostTextHTML etc.
}

// LoadHTMLPage reads and parses an HTML file from the given path
func LoadHTMLPage(filePath string) (*HTMLPage, error) {
	// Verify file exists and is readable
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open HTML file %s: %w", filePath, err)
	}
	defer file.Close()

	// Read the file content
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML file %s: %w", filePath, err)
	}

	// Parse the HTML content using goquery from the bytes read, not the original file reader
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(contentBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML content from %s: %w", filePath, err)
	}

	return &HTMLPage{
		FilePath: filePath,
		Content:  doc,
	}, nil
}

// GetPostBlocks identifies and returns the HTML blocks containing individual posts
// Each block is a goquery.Selection representing the <tr> element of a post.
func (p *HTMLPage) GetPostBlocks() ([]PostBlock, error) {
	var blocks []PostBlock

	// Find all tables with class "normal" that are direct children of "div#container".
	// Then, select the second one (index 1), which is expected to be the posts table.
	postsTable := p.Content.Find("body > div#container > table.normal").Eq(1)

	if postsTable.Length() == 0 {
		// This means the second table.normal was not found. Could be an unexpected page structure.
		// Return empty blocks, let caller decide if it's an error.
		return blocks, nil
	}

	// Now, within this specific table, find the <tr> elements that are actual posts.
	// These <tr> elements must contain the characteristic <td> cells for user info and post content.
	postRowSelector := "tr:has(td.normal.bgc1.c.w13.vat):has(td.normal.bgc1.vat.w90)"

	postsTable.Find(postRowSelector).Each(func(i int, s *goquery.Selection) {
		blocks = append(blocks, PostBlock{Selection: s})
	})

	// AC5: Handle pages with multiple posts (covered by Find().Each())
	// AC6: Provide access to isolated HTML (PostBlock.Selection provides this)

	// If no blocks were found, it might not be an error, but could be an empty page or different structure.
	// The calling code can decide how to handle zero blocks based on context (e.g., log it as per AC7).
	return blocks, nil
} 