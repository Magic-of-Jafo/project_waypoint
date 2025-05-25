# Story 1.1: Develop Sub-Forum Page Navigation Logic

## Status: Done

## Story

- As a Developer of the Indexing System,
- I want to create a module that can reliably identify all unique pagination links for a given Magic Cafe sub-forum's thread listing pages (e.g., `viewforum.php?forum=X`) and generate an ordered sequence of all page URLs for that sub-forum,
- So that the Indexing System can systematically visit every page where individual topic/thread URLs are listed, ensuring no page of threads is missed.

## Acceptance Criteria (ACs)

1.  Given the base URL of any Magic Cafe sub-forum (e.g., `https://www.themagiccafe.com/forums/viewforum.php?forum=X`), the system can successfully fetch and parse its HTML content.
2.  The system can correctly identify the HTML elements containing pagination links on a sub-forum's thread listing page (e.g., links for "Next", "Previous", and specific page numbers).
3.  If a "Next Page" link exists, the system can extract its URL.
4.  If numbered page links exist (e.g., "1, 2, 3, ..., Last"), the system can extract the URL for each explicitly listed page number.
5.  The system can determine the URL for the last page of thread listings, either by finding a "Last Page" link or by deducing it from the highest numbered page link present.
6.  The system can generate a complete, ordered list of all unique URLs for every page of thread listings within that sub-forum, from page 1 to the last page.
7.  The logic correctly handles sub-forums that have only a single page of threads (i.e., no pagination links are present beyond the current page).
8.  The logic correctly handles sub-forums with multiple pages of threads, including scenarios where pagination might involve ellipses (e.g., "1, 2, ..., 10, 11").
9.  The system logs an informative message or error if it encounters an unexpected pagination structure it cannot parse on a sub-forum page.

## Tasks / Subtasks

- [x] Task 1 (AC: 1, 2) Design and implement HTML fetching and initial parsing for a sub-forum page.
  - [x] Subtask 1.1: Function to fetch HTML content given a URL.
  - [x] Subtask 1.2: Basic HTML parsing to identify potential pagination areas.
- [x] Task 2 (AC: 3, 4, 5) Implement logic to identify and extract all types of pagination links (Next, Numbered, Last).
  - [x] Subtask 2.1: Extract "Next Page" URL.
  - [x] Subtask 2.2: Extract all numbered page URLs.
  - [x] Subtask 2.3: Determine "Last Page" URL (either direct or by highest number).
- [x] Task 3 (AC: 6) Implement logic to consolidate extracted links into a complete, ordered list of unique page URLs for the sub-forum.
- [x] Task 4 (AC: 7, 8) Test logic with single-page, multi-page, and ellipsis-style pagination.
- [x] Task 5 (AC: 9) Implement error logging for unparsable pagination structures.

## Dev Technical Guidance

-   Initial focus should be on robustly handling the known pagination structures of The Magic Cafe.
-   Consider using a well-established HTML parsing library (e.g., BeautifulSoup in Python).
-   Ensure polite scraping (delay between requests) is handled by the calling orchestrator or a higher-level module, not necessarily within this specific navigation logic, though it should be compatible.

## Story Progress Notes

### Agent Model Used: `Gemini 2.5 Pro (via Cursor)`

### Completion Notes List
- Implemented `FetchHTML` and `ParsePaginationLinks` functions in Go.
- `ParsePaginationLinks` correctly identifies all page URLs from the main sub-forum pagination element.
- Assumes a fixed number of topics per page (30) to reconstruct the full list of pages when ellipses are present.
- Basic command-line application `cmd/indexer/main.go` created to demonstrate functionality.
- All acceptance criteria are met.

### Change Log 