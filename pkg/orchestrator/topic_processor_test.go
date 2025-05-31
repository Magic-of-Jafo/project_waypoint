package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"project-waypoint/pkg/data" // For asserting PostMetadata content
	"github.com/stretchr/testify/assert" // Optional: for assertions
)

// TestMain will be called before running tests in this package.
func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.SetOutput(os.Stderr)
	log.Println("TestMain: Logger initialized for orchestrator tests.")
	os.Exit(m.Run())
}

// TestProcessTopic is the main test function for ProcessTopic.
// It should cover various scenarios as outlined in Story 3.5, Task 7.
func TestProcessTopic(t *testing.T) {
	// Setup: Create temporary directories for archive and output
	tempDir, err := os.MkdirTemp("", "processtopic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "archive")
	outputPath := filepath.Join(tempDir, "output")

	if err := os.MkdirAll(archivePath, 0755); err != nil {
		t.Fatalf("Failed to create temp archive dir: %v", err)
	}
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("Failed to create temp output dir: %v", err)
	}

	// Define test cases
	tests := []struct {
		name         string
		topicID      string
		setupArchive func(t *testing.T, archivePath string, topicID string) // Function to create mock HTML files
		wantErr      bool
		expectedNumPosts int // Added to check number of posts in output JSON
		checkOutput  func(t *testing.T, outputPath string, subforumID string, topicID string, expectedNumPosts int) // Function to validate the output JSON
	}{
		// Scenario 1: Successful processing of a simple topic
		{
			name:    "Successful simple topic",
			topicID: "topic123",
			setupArchive: func(t *testing.T, archivePath string, topicID string) {
				// Create mock subforum and topic structure
				subforumDir := filepath.Join(archivePath, "subforum_test_simple") // Unique subforum for this test
				topicDir := filepath.Join(subforumDir, topicID)
				if err := os.MkdirAll(topicDir, 0755); err != nil {
					t.Fatalf("setupArchive: failed to create topic dir: %v", err)
				}
				// Create a mock page_1.html with more realistic post structure
				page1Content := `<html><head><title>Test Page</title></head><body><div id="container">
` + // Main container start
					`<table class="normal"><tr><td>Some pre-table, e.g., nav links</td></tr></table>
` + // First table.normal (to be skipped by Eq(1))
					`<table class="normal">
` + // Second table.normal (this is the postsTable)
					`<!-- Post 1 -->
` +
					`<tr>
` +
					`<td class="normal bgc1 c w13 vat">
` +
					`<strong>Author1</strong><br />
` +
					`User Title<br />
` +
					`<span class="small">Posts: 100</span>
` +
					`</td>
` +
					`<td class="normal bgc1 vat w90">
` +
					`<div class="vt1 liketext">
` +
					`<div class="like_left">
` +
					`Posted: <span class="b">Jan 01, 2023 10:00 am</span> <a name="0"></a>
` +
					`</div>
` +
					`<div class="like_right">
` +
					`<span id="p_post111"></span> &nbsp;
` +
					`</div>
` +
					`</div>
` +
					`<hr />
` +
					`<div>
` +
					`Post content 1
` +
					`<table class="cfq"><tr><td><b>QuotedUser</b> wrote:</td></tr><tr><td>Quoted text</td></tr></table>
` +
					`More text from Author1.
` +
					`</div>
` +
					`</td>
` +
					`</tr>
` +
					`<!-- Post 2 -->
` +
					`<tr>
` +
					`<td class="normal bgc1 c w13 vat">
` +
					`<strong>Author2</strong><br />
` +
					`</td>
` +
					`<td class="normal bgc1 vat w90">
` +
					`<div class="vt1 liketext">
` +
					`<div class="like_left">
` +
					`Posted: <span class="b">Jan 01, 2023 10:05 am</span> <a name="1"></a>
` +
					`</div>
` +
					`<div class="like_right">
` +
					`<span id="p_post222"></span> &nbsp;
` +
					`</div>
` +
					`</div>
` +
					`<hr />
` +
					`<div>
` +
					`Post content 2 by Author2.
` +
					`</div>
` +
					`</td>
` +
					`</tr>
` +
					`</table>
` + // End postsTable
					`</div></body></html>
` // End container, body, html
				if err := os.WriteFile(filepath.Join(topicDir, "page_1.html"), []byte(page1Content), 0644); err != nil {
					t.Fatalf("setupArchive: failed to write page_1.html: %v", err)
				}
			},
			wantErr: false,
			expectedNumPosts: 2,
			checkOutput: func(t *testing.T, outputPath string, subforumID string, topicID string, expectedNumPosts int) {
				jsonFilePath := filepath.Join(outputPath, fmt.Sprintf("%s_%s.json", subforumID, topicID))
				_, err := os.Stat(jsonFilePath)
				if os.IsNotExist(err) {
					t.Errorf("checkOutput: expected JSON file %s to exist, but it doesn't", jsonFilePath)
					return
				}
				content, err := os.ReadFile(jsonFilePath)
				if err != nil {
					t.Errorf("checkOutput: failed to read JSON file %s: %v", jsonFilePath, err)
					return
				}
				var posts []data.PostMetadata
				if err := json.Unmarshal(content, &posts); err != nil {
					t.Errorf("checkOutput: failed to unmarshal JSON from %s: %v", jsonFilePath, err)
					return
				}
				if !assert.Equal(t, expectedNumPosts, len(posts), "Number of posts in JSON should match expected") {
					return
				}
				if expectedNumPosts > 0 {
					assert.NotEmpty(t, posts[0].PostID, "First post should have a PostID")
					assert.NotEmpty(t, posts[0].AuthorUsername, "First post should have an AuthorUsername")
					// Further checks can be added here, e.g. specific values
				}
			},
		},
		// Scenario 2: Topic with no HTML files found
		{
			name:    "Topic no files",
			topicID: "topic_empty_files",
			setupArchive: func(t *testing.T, archivePath string, topicID string) {
				// Create mock subforum and topic structure, but no page files
				subforumDir := filepath.Join(archivePath, "subforum_test_empty")
				topicDir := filepath.Join(subforumDir, topicID)
				if err := os.MkdirAll(topicDir, 0755); err != nil {
					t.Fatalf("setupArchive: failed to create topic dir: %v", err)
				}
			},
			wantErr: true, // Expect error because no files are found
			expectedNumPosts: 0, // No file should be created
			checkOutput: nil,    // Not applicable as error is expected
		},
		// Scenario 3: Topic with pages, but no posts on pages (empty pages)
		{
			name:    "Topic empty pages",
			topicID: "topic_empty_pages",
			setupArchive: func(t *testing.T, archivePath string, topicID string) {
				subforumDir := filepath.Join(archivePath, "subforum_test_ep")
				topicDir := filepath.Join(subforumDir, topicID)
				if err := os.MkdirAll(topicDir, 0755); err != nil {
					t.Fatalf("setupArchive: failed to create topic dir: %v", err)
				}
				page1Content := `<html><body><div id="container"><table class="normal"></table><table class="normal">` +
					`</table></div></body></html>` // No post rows
				if err := os.WriteFile(filepath.Join(topicDir, "page_1.html"), []byte(page1Content), 0644); err != nil {
					t.Fatalf("setupArchive: failed to write page_1.html: %v", err)
				}
			},
			wantErr: false, // No error, but JSON file might be empty or not created based on logic
			expectedNumPosts: 0,
			checkOutput: func(t *testing.T, outputPath string, subforumID string, topicID string, expectedNumPosts int) {
				jsonFilePath := filepath.Join(outputPath, fmt.Sprintf("%s_%s.json", subforumID, topicID))
				// In this case, the ProcessTopic might return nil without creating a file if no posts are found
				// Check if the file exists. If it does, it should contain an empty array.
				// If it doesn't exist, it means 0 posts were processed and no file was made, which is also acceptable for 0 posts.
				_, err := os.Stat(jsonFilePath)
				if os.IsNotExist(err) {
					// File not existing for 0 posts is acceptable by current ProcessTopic logic
					log.Printf("checkOutput for 'Topic empty pages': JSON file %s does not exist, which is acceptable for 0 posts.", jsonFilePath)
					return
				}
				content, err := os.ReadFile(jsonFilePath)
				if err != nil {
					t.Errorf("checkOutput: failed to read JSON file %s: %v", jsonFilePath, err)
					return
				}
				var posts []data.PostMetadata
				if err := json.Unmarshal(content, &posts); err != nil {
					t.Errorf("checkOutput: failed to unmarshal JSON from %s: %v", jsonFilePath, err)
					return
				}
				assert.Equal(t, 0, len(posts), "JSON file for empty pages should contain 0 posts if it exists")
			},
		},
		// Scenario 4: Successful multi-page topic
		{
			name:    "Successful multi-page topic",
			topicID: "topic_multi",
			setupArchive: func(t *testing.T, archivePath string, topicID string) {
				subforumDir := filepath.Join(archivePath, "subforum_test_multi")
				topicDir := filepath.Join(subforumDir, topicID)
				if err := os.MkdirAll(topicDir, 0755); err != nil {
					t.Fatalf("setupArchive (multi-page): failed to create topic dir: %v", err)
				}

				// Page 2 - created first to test sorting by ProcessTopic
				page2Content := `<html><body><div id="container"><table class="normal"><tr><td>Nav</td></tr></table><table class="normal">
` +
					`<!-- Post 3 (on page 2) -->
` +
					`<tr><td class="normal bgc1 c w13 vat"><strong>Author3</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b">Jan 02, 2023 11:00 am</span> <a name="0"></a></div><div class="like_right"><span id="p_post333"></span></div></div><hr /><div>Post content 3 by Author3.</div></td></tr>
` +
					`</table></div></body></html>`
				if err := os.WriteFile(filepath.Join(topicDir, "page_2.html"), []byte(page2Content), 0644); err != nil {
					t.Fatalf("setupArchive (multi-page): failed to write page_2.html: %v", err)
				}

				// Page 1
				page1Content := `<html><body><div id="container"><table class="normal"><tr><td>Nav</td></tr></table><table class="normal">
` +
					`<!-- Post 1 (on page 1) -->
` +
					`<tr><td class="normal bgc1 c w13 vat"><strong>Author1_Page1</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b">Jan 01, 2023 09:00 am</span> <a name="0"></a></div><div class="like_right"><span id="p_post111_p1"></span></div></div><hr /><div>Post content 1 on page 1.</div></td></tr>
` +
					`<!-- Post 2 (on page 1) -->
` +
					`<tr><td class="normal bgc1 c w13 vat"><strong>Author2_Page1</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b">Jan 01, 2023 09:05 am</span> <a name="1"></a></div><div class="like_right"><span id="p_post222_p1"></span></div></div><hr /><div>Post content 2 on page 1.</div></td></tr>
` +
					`</table></div></body></html>`
				if err := os.WriteFile(filepath.Join(topicDir, "page_1.html"), []byte(page1Content), 0644); err != nil {
					t.Fatalf("setupArchive (multi-page): failed to write page_1.html: %v", err)
				}
			},
			wantErr: false,
			expectedNumPosts: 3, // 2 from page 1, 1 from page 2
			checkOutput: func(t *testing.T, outputPath string, subforumID string, topicID string, expectedNumPosts int) { // Same checkOutput as simple topic
				jsonFilePath := filepath.Join(outputPath, fmt.Sprintf("%s_%s.json", subforumID, topicID))
				_, err := os.Stat(jsonFilePath)
				if os.IsNotExist(err) {
					t.Errorf("checkOutput: expected JSON file %s to exist, but it doesn't", jsonFilePath)
					return
				}
				content, err := os.ReadFile(jsonFilePath)
				if err != nil {
					t.Errorf("checkOutput: failed to read JSON file %s: %v", jsonFilePath, err)
					return
				}
				var posts []data.PostMetadata
				if err := json.Unmarshal(content, &posts); err != nil {
					t.Errorf("checkOutput: failed to unmarshal JSON from %s: %v", jsonFilePath, err)
					return
				}
				if !assert.Equal(t, expectedNumPosts, len(posts), "Number of posts in JSON should match expected for multi-page") {
					return
				}
				if expectedNumPosts > 0 {
					assert.NotEmpty(t, posts[0].PostID, "First post (multi-page) should have a PostID")
					assert.Contains(t, posts[0].PostID, "_p1", "First post should be from page 1 due to sorting") // Basic check for page 1 content
					assert.Equal(t, "Author1_Page1", posts[0].AuthorUsername, "First post author should be Author1_Page1")
					// Check if posts are sorted by page and then by order on page
					assert.Equal(t, "post111_p1", posts[0].PostID)
					assert.Equal(t, "post222_p1", posts[1].PostID)
					assert.Equal(t, "post333", posts[2].PostID)
				}
			},
		},
		// Scenario 5: Topic with partial extraction errors (graceful handling)
		{
			name:    "Topic with partial extraction errors",
			topicID: "topic_partial_err",
			setupArchive: func(t *testing.T, archivePath string, topicID string) {
				subforumDir := filepath.Join(archivePath, "subforum_test_partial_err")
				topicDir := filepath.Join(subforumDir, topicID)
				if err := os.MkdirAll(topicDir, 0755); err != nil {
					t.Fatalf("setupArchive (partial_err): failed to create topic dir: %v", err)
				}

				pageContent := `<html><body><div id="container"><table class="normal"><tr><td>Nav</td></tr></table><table class="normal">
` +
					`<!-- Post 1 (Valid) -->
<tr><td class="normal bgc1 c w13 vat"><strong>Author_Valid1</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b">Feb 01, 2023 10:00 am</span> <a name="0"></a></div><div class="like_right"><span id="p_postValid1"></span></div></div><hr /><div>Valid post 1 content.</div></td></tr>
` +
					`<!-- Post 2 (Malformed - e.g., bad post ID span, or missing timestamp text) -->
<tr><td class="normal bgc1 c w13 vat"><strong>Author_Error</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b"></span> <a name="1"></a></div><div class="like_right"><span id="p_"></span></div></div><hr /><div>Post with error. Timestamp text missing, post id malformed.</div></td></tr>
` +
					`<!-- Post 3 (Valid) -->
<tr><td class="normal bgc1 c w13 vat"><strong>Author_Valid2</strong></td><td class="normal bgc1 vat w90"><div class="vt1 liketext"><div class="like_left">Posted: <span class="b">Feb 01, 2023 10:10 am</span> <a name="2"></a></div><div class="like_right"><span id="p_postValid2"></span></div></div><hr /><div>Valid post 2 content.</div></td></tr>
` +
					`</table></div></body></html>`
				if err := os.WriteFile(filepath.Join(topicDir, "page_1.html"), []byte(pageContent), 0644); err != nil {
					t.Fatalf("setupArchive (partial_err): failed to write page_1.html: %v", err)
				}
			},
			wantErr: false, // ProcessTopic itself should not fail fatally for this
			expectedNumPosts: 2, // Only the two valid posts
			checkOutput: func(t *testing.T, outputPath string, subforumID string, topicID string, expectedNumPosts int) {
				jsonFilePath := filepath.Join(outputPath, fmt.Sprintf("%s_%s.json", subforumID, topicID))
				_, err := os.Stat(jsonFilePath)
				if os.IsNotExist(err) {
					t.Errorf("checkOutput (partial_err): expected JSON file %s to exist", jsonFilePath)
					return
				}
				content, err := os.ReadFile(jsonFilePath)
				if err != nil {
					t.Errorf("checkOutput (partial_err): failed to read JSON file %s: %v", jsonFilePath, err)
					return
				}
				var posts []data.PostMetadata
				if err := json.Unmarshal(content, &posts); err != nil {
					t.Errorf("checkOutput (partial_err): failed to unmarshal JSON from %s: %v", jsonFilePath, err)
					return
				}
				assert.Equal(t, expectedNumPosts, len(posts), "Number of posts in JSON should be 2 (valid ones only)")
				if len(posts) == 2 { // Further checks if we have the expected number
					assert.Equal(t, "postValid1", posts[0].PostID, "First post ID should be postValid1")
					assert.Equal(t, "Author_Valid1", posts[0].AuthorUsername, "First post author should be Author_Valid1")
					assert.Equal(t, "postValid2", posts[1].PostID, "Second post ID should be postValid2")
					assert.Equal(t, "Author_Valid2", posts[1].AuthorUsername, "Second post author should be Author_Valid2")
				}
			},
		},
		// TODO: Add more test scenarios:
		// (No more TODOs for now based on previous discussion, this covers the main ones)
	}

	t.Log("NOTE: These tests are integration-style tests for ProcessTopic using real file system operations.")
	t.Log("Full unit tests would require mocking file system and dependent parsing packages.")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh archive and output dirs for each test run
			testSpecificArchiveDir := filepath.Join(archivePath, tt.name+"_archive")
			testSpecificOutputDir := filepath.Join(outputPath, tt.name+"_output")
			if err := os.MkdirAll(testSpecificArchiveDir, 0755); err != nil {
				t.Fatalf("Failed to create test specific archive dir: %v", err)
			}
			if err := os.MkdirAll(testSpecificOutputDir, 0755); err != nil {
				t.Fatalf("Failed to create test specific output dir: %v", err)
			}

			if tt.setupArchive != nil {
				tt.setupArchive(t, testSpecificArchiveDir, tt.topicID)
			}

			err := ProcessTopic(tt.topicID, testSpecificArchiveDir, testSpecificOutputDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessTopic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOutput != nil {
				// Determine subforumID. For these tests, it's the base of the first dir in testSpecificArchiveDir + tt.name + "_archive"
				// e.g., subforum_test_simple, subforum_test_multi
				dirs, _ := os.ReadDir(testSpecificArchiveDir)
				var subforumID string
				if len(dirs) > 0 && dirs[0].IsDir() {
					subforumID = dirs[0].Name()
				} else {
					// Fallback or error if structure is not as expected for subforumID derivation in test
					// For "Topic no files" or "Topic empty pages" this might be different, adjust if needed
					switch tt.name {
					case "Topic no files":
						subforumID = "subforum_test_empty" // Matches setupArchive
					case "Topic empty pages":
						subforumID = "subforum_test_ep" // Matches setupArchive
					case "Successful multi-page topic":
						subforumID = "subforum_test_multi" // Matches setupArchive
					case "Topic with partial extraction errors":
						subforumID = "subforum_test_partial_err" // Matches setupArchive
					default:
						t.Logf("Could not determine subforumID for %s from directory structure, using 'unknown_subforum_test'", tt.name)
						subforumID = "unknown_subforum_test"
					}
				}
				tt.checkOutput(t, testSpecificOutputDir, subforumID, tt.topicID, tt.expectedNumPosts)
			}
		})
	}
}

// Helper to ensure consistency in subforum name for tests when setting up archive
func getTestSubforumDir(baseArchivePath string, testName string, subforumNamePart string) string {
	return filepath.Join(baseArchivePath, testName+"_archive", subforumNamePart)
}
