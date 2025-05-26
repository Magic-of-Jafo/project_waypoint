# Project Waypoint - "Waypoint Archive" Scripts: Project Structure

**Date:** May 26, 2025
**Version:** 1.0

## 1. Overview

This document outlines the recommended project directory structure for the "Waypoint Archive" scripts of Project Waypoint. These scripts are primarily developed in Go.

## 2. Root Directory: `waypoint_archive_scripts/`

All Go modules, source code, documentation, and supporting files for the archival scripts will reside under this main project directory.

## 3. Proposed Directory Structure

waypoint_archive_scripts/
├── cmd/                          # Main applications (executables)
│   ├── indexer/                  # Main package and entry point for the Indexing script (Epic 1)
│   │   └── main.go
│   ├── archiver/                 # Main package and entry point for the Archival script (Epic 2)
│   │   └── main.go
│   └── extractor/                # Main package and entry point for the Data Extractor script (Epic 3)
│       └── main.go
├── pkg/                          # Shared library code (internal packages)
│   ├── config/                   # Configuration loading and management
│   │   └── config.go
│   ├── httpclient/               # Wrapper or utilities for HTTP requests (if needed beyond std lib)
│   │   └── client.go
│   ├── indexerlogic/             # Core logic for indexing (Story 1.1-1.5)
│   │   └── indexer.go
│   ├── archiverlogic/            # Core logic for HTML archival (Story 2.1-2.7)
│   │   └── archiver.go
│   ├── parser/                   # HTML parsing utilities (used by indexer, archiver, extractor)
│   │   └── parser.go
│   ├── utils/                    # Common utility functions (logging helpers, file helpers, etc.)
│   │   └── utils.go
│   └── data/                     # Struct definitions for shared data types (e.g., TopicInfo, PostData)
│       └── types.go
├── docs/                         # Project documentation
│   ├── PRD_Project_Waypoint.md
│   ├── operational-guidelines.md
│   ├── Waypoint_Archive_Tech_Stack.md
│   ├── project-structure.md      # This file
│   └── epics/
│       ├── epic-1.md             # (If created - detailed stories for Epic 1)
│       ├── epic-2.md             # (To be created - detailed stories for Epic 2)
│       └── epic-3.md             # (To be created - detailed stories for Epic 3)
├── testdata/                     # Sample HTML files or other data for unit/integration tests
│   └── sample_forum_page.html
├── scripts/                      # Utility shell scripts (e.g., for building, running, testing)
│   └── run_indexer.sh
├── .gitignore                    # Specifies intentionally untracked files that Git should ignore
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
└── README.md                     # Project overview, setup, and usage instructions for scripts

## 4. Key Directory Explanations

* **`cmd/`**: Contains the main application packages. Each subdirectory (e.g., `cmd/indexer/`) will have a `main.go` file that serves as the entry point for an executable script.
* **`pkg/`**: Contains library code that's shared and reusable across the different main applications or within a larger application. This promotes modularity. For instance:
    * `pkg/config`: Handles loading configuration (e.g., from files or environment variables).
    * `pkg/indexerlogic`: Core logic for Epic 1.
    * `pkg/archiverlogic`: Core logic for Epic 2.
    * `pkg/parser`: Common HTML parsing functions.
    * `pkg/utils`: General helper functions.
    * `pkg/data`: Go struct definitions for data passed between modules.
* **`docs/`**: All project documentation, including this file, the PRD, Operational Guidelines, Tech Stack, and detailed Epic breakdowns.
* **`testdata/`**: Stores any static files needed for running tests (e.g., sample HTML snippets).
* **`scripts/`**: Optional directory for helper shell scripts for common tasks.

## 5. Go Package Conventions

* Code within `pkg/` subdirectories should belong to a package named after the directory (e.g., code in `pkg/config/` is `package config`).
* Code within `cmd/` subdirectories is typically `package main`.

## 6. Test Files

* Test files (`_test.go`) should be co-located with the code they are testing within the same package directory, as per Go conventions.