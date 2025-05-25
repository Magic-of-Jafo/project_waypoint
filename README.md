# Project Waypoint

Project Waypoint is a Go-based application designed to crawl and index web forum content. The initial version focuses on discovering all paginated page URLs within a specific sub-forum of The Magic Cafe (`themagiccafe.com`).

## Features

*   **Sub-Forum Page Discovery:** Given the URL of a sub-forum page on The Magic Cafe, the `indexer` tool can:
    *   Fetch the HTML content of the page.
    *   Parse pagination links (including numbered pages and those with ellipses).
    *   Generate a complete list of all unique URLs for every page within that sub-forum's thread listing.

## Technology Stack

*   **Go (Golang):** The primary programming language.
*   **`github.com/PuerkitoBio/goquery`:** Used for HTML parsing and DOM manipulation.

## Project Structure

```
project-waypoint/
├── cmd/
│   └── indexer/
│       └── main.go         # Main application for the indexer CLI
├── internal/
│   └── indexer/
│       └── navigation/
│           ├── navigation.go       # Core logic for fetching and parsing pagination
│           └── navigation_test.go  # Unit tests for navigation logic
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── docs/                   # Project documentation (stories, PRD, etc.)
└── README.md               # This file
```

## Setup

1.  **Clone the repository (if you haven't already):**
    ```bash
    git clone <your-repository-url>
    cd project-waypoint
    ```
2.  **Ensure Go is installed:**
    This project was developed with Go. If you don't have it installed, download it from [golang.org/dl](https://golang.org/dl/).

## Usage

The primary component currently is the `indexer` command-line tool.

1.  **Build the indexer:**
    Navigate to the project root directory and run:
    ```bash
    go build ./cmd/indexer/main.go
    ```
    This will create an executable named `main` (or `main.exe` on Windows) in the project root. You might want to rename it or build it directly into a `bin` directory. A more common build pattern for the `indexer` in its own directory would be:
    ```bash
    cd cmd/indexer
    go build .
    ```
    This will create `indexer` (or `indexer.exe`) inside `cmd/indexer/`.

2.  **Run the indexer:**
    Execute the built program, providing the starting URL of a Magic Cafe sub-forum as a command-line argument.

    If you built in `cmd/indexer/`:
    ```bash
    ./cmd/indexer/indexer "https://www.themagiccafe.com/forums/viewforum.php?forum=54"
    ```
    (On Windows, use `.\cmd\indexer\indexer.exe "..."`)

    The tool will then output a list of all discovered page URLs for that sub-forum.

## Example Output

```
2023/10/27 10:00:00 Starting sub-forum page indexer...
2023/10/27 10:00:00 Fetching initial page: https://www.themagiccafe.com/forums/viewforum.php?forum=54
2023/10/27 10:00:01 Parsing pagination links...
2023/10/27 10:00:01 Discovered 64 page URLs for sub-forum:
1: https://www.themagiccafe.com/forums/viewforum.php?forum=54
2: https://www.themagiccafe.com/forums/viewforum.php?forum=54&start=30
3: https://www.themagiccafe.com/forums/viewforum.php?forum=54&start=60
...
64: https://www.themagiccafe.com/forums/viewforum.php?forum=54&start=1890
2023/10/27 10:00:01 Sub-forum page indexing complete.
```

## Future Development

(Placeholder for future features and enhancements)

## Contributing

(Placeholder for contribution guidelines)
