Okay, I understand! You want the complete, detailed versions of the User Stories we drafted, including all the Acceptance Criteria for each, within that Epic 3 document. That makes perfect sense for a handoff to your Scrum Master and development team – more detail is always better for implementation!

Let's regenerate the docs/epics/epic-3.md document with the full content of each story.

--- START OF FILE docs/epics/epic-3.md ---

# Epic 3: Structured Data Extraction System Development & Initial Processing

**Goal:** To develop, test, and execute a system that processes the archived raw HTML (created in Epic 2), accurately extracting post metadata and content (including structured quote information) into a locally stored, machine-readable JSON format (one file per topic), making the data ready for future AI processing (Epic 4).

**Dependency:** Requires the successful completion of Epic 2 (Raw HTML Archival System) and access to the archived raw HTML files stored according to Epic 2's defined structure. Also relies on the Topic Index generated in Epic 1.

---

## Defined Artifacts

**Target Structured Data Format (JSON Schema for a single Post):**

The structured data extraction process will produce JSON files, with each file representing a single topic. The content of these files will be an array of post objects, where each post object conforms to the following structure:

```json
{
  "post_id": "unique_id_from_span_id", // Extracted from HTML attribute (e.g., the number from id="p_4174478")
  "topic_id": "the_topic_id_this_post_belongs_to", // Derived from archive file path or Epic 1 index
  "subforum_id": "the_subforum_id_this_post_belongs_to", // Derived from archive file path or Epic 1 index
  "page_number": 1, // Derived from archive file path or URL
  "post_order_on_page": 0, // Extracted from HTML attribute (e.g., the number from <a name="0">)
  "post_url": "url_to_this_specific_post_if_available_or_page_url", // Can be constructed
  "author_username": "Username", // Extracted from post HTML
  "timestamp": "YYYY-MM-DD HH:MM:SS", // Parsed timestamp from post HTML
  "content_blocks": [ // Array of content blocks preserving original order
    {
      "type": "new_text",
      "content": "This is clean text from the user's post, with HTML/BBCode tags removed but emojis represented and line breaks preserved."
    },
    {
      "type": "quote",
      "quoted_user": "QuotedUsername", // Extracted from quote attribution
      "quoted_timestamp": "YYYY-MM-DD HH:MM:SS", // Parsed timestamp from quote attribution (if available)
      "quoted_text": "This is the clean text of the quoted content, with its own internal HTML/BBCode tags removed."
    }
    // More blocks as they appear in the original post
  ]
}


Structured Data Filename Convention:

Each topic's structured data will be saved as a single JSON file using the following naming convention:

{subforum_id}_{topic_id}.json

These files should be saved to a user-configurable output directory, ideally on the Synology NAS.

User Stories

The following User Stories detail the requirements for developing the Structured Data Extraction System for Epic 3.

Story 3.1: Read Archived HTML and Identify Posts

As the Structured Data Extraction System,

I want to iterate through the raw HTML files stored in the local "Waypoint Archive" (organized as per Epic 2), load the content of each page, and reliably identify the distinct HTML block corresponding to each individual post within that page,

So that I can then process each post separately to extract its metadata and content.

Acceptance Criteria (ACs):

AC1: The system MUST be configurable with the root file path of the local "Waypoint Archive" created by Epic 2.

AC2: The system MUST be able to navigate the directory structure within the archive root (e.g., {ARCHIVE_ROOT}/{sub_forum_id_or_name}/{topic_id}/page_{page_number}.html).

AC3: The system MUST successfully read the raw HTML content from each .html file found within the archive structure.

AC4: For each loaded HTML page, the system MUST employ robust parsing techniques to identify and isolate the specific HTML section or element that contains the data for a single forum post.

AC5: The system MUST be able to handle pages with multiple posts and correctly identify each individual post block on the page.

AC6: The system MUST provide access to the isolated HTML content for each identified post block for subsequent processing steps (like metadata and content extraction).

AC7: The system MUST log the file path of each HTML page being processed and the number of post blocks successfully identified within that page.

AC8: The system MUST implement error handling for scenarios like unreadable files, missing files, or parsing errors, logging these issues and ideally continuing processing with other files where possible (as per docs/operational-guidelines.md).

Story 3.2: Extract Core Post Metadata

As the Structured Data Extraction System,

I want to take the isolated HTML block for a single post (identified in Story 3.1) and reliably extract its core metadata fields: post_id, topic_id, subforum_id, page_number, post_order_on_page, author_username, and timestamp,

So that this essential identifying and contextual information is available for inclusion in the final structured JSON output for that post.

Acceptance Criteria (ACs):

AC1: Given the HTML content corresponding to a single post block, the system MUST accurately extract the author_username.

AC2: The system MUST accurately extract and parse the timestamp of the post into a consistent format (e.g., "YYYY-MM-DD HH:MM:SS") from the post's HTML.

AC3: The system MUST accurately extract the post_id using the value found in the id attribute of the relevant HTML element (e.g., the number from id="p_4174478").

AC4: The system MUST accurately extract the post_order_on_page using the value found in the name attribute of the relevant HTML anchor tag (e.g., the number from <a name="0"></a>).

AC5: The system MUST determine the topic_id for the post. This might be extracted from the original filename of the HTML page or passed in from the process orchestrating the parsing (linking back to Epic 2's file structure).

AC6: The system MUST determine the subforum_id for the post. Similar to topic_id, this is likely derived from the filename, file path, or passed in context from the Epic 2 archive structure.

AC7: The system MUST determine the page_number for the post. This is also likely derived from the filename or passed in context (linking back to Epic 2's file structure).

AC8: The system MUST handle variations in HTML structure or missing elements for these metadata fields gracefully, logging warnings or errors but attempting to extract as much information as possible and ideally marking the specific post as having potential metadata extraction issues rather than halting the entire process.

AC9: The extracted metadata fields for a post MUST be made available in a structured format (e.g., a map or object in Go/Python) to be used in subsequent steps (like assembling the final JSON).

Story 3.3 (Combined): Parse Post Content into Structured Blocks and Extract Quote Details

As the Structured Data Extraction System,

I want to parse the main HTML content area of a single post and identify sequences of the author's new_text and distinct quote blocks, and for each quote block found, extract the quoted_user, quoted_timestamp (if available), and quoted_text,

So that the post's complete content, including structured quote information, is available as an ordered list of blocks for the final JSON output.

Acceptance Criteria (ACs):

AC1: Given the HTML content area of a single post, the system MUST be able to identify and separate segments of HTML that represent the author's direct new_text.

AC2: The system MUST be able to identify and separate segments of HTML that represent quoted content, recognizing the specific HTML structure used for quotes on The Magic Cafe forum.

AC3: The system MUST maintain the correct order of new_text and quote blocks as they appear sequentially in the original HTML content.

AC4: For each identified quote block, the system MUST accurately identify and extract the username of the person being quoted (quoted_user).

AC5: For each identified quote block, the system MUST attempt to identify and extract a timestamp associated with the quote within the quote block's HTML. If present and distinguishable, it MUST be parsed into a consistent format (quoted_timestamp). If not found, this field should be null/empty.

AC6: For each identified quote block, the system MUST accurately extract the main text content of the quote block, excluding the attribution lines parsed in AC4 and AC5. This content will be the quoted_text.

AC7: The system MUST be able to handle variations in quote attribution formats to extract quoted_user and quoted_timestamp robustly.

AC8: The system MUST be robust to potentially nested quotes or other complex HTML structures within the post content area, ensuring correct segmentation and quote detail extraction.

AC9: The system MUST represent the parsed content as an ordered list or array of block structures. For new_text blocks, the structure will contain the text content. For quote blocks, the structure MUST include the type (quote) and the extracted quoted_user, quoted_timestamp, and quoted_text.

AC10: The system MUST log any parsing errors encountered within a post's content area (for both segmentation and quote detail extraction), ideally indicating the problematic post and block but continuing to process other posts if possible.

Story 3.4: Extract Clean Text from New Text Blocks

As the Structured Data Extraction System,

I want to take the raw HTML content identified as a new_text block and convert it into clean, plain text, removing most HTML tags and unwanted formatting, but specifically preserving a representation of embedded emojis and essential readability elements like line breaks,

So that the author's original contribution, including key non-textual elements like emojis, is accurately represented as a simple string within the structured JSON output.

Acceptance Criteria (ACs):

AC1: Given the raw HTML content of a new_text block, the system MUST remove standard HTML tags.

AC2: When encountering an <img> tag (commonly used for emojis), the system MUST attempt to extract the text from the alt or title attribute. If successful, this extracted text (e.g., ":)" or "Smile") SHOULD be inserted into the plain text output at the location of the image tag. If no alt or title is found, a placeholder (e.g., [image]) may be inserted, or the tag removed, depending on complexity/feasibility.

AC3: The system MUST preserve essential formatting like line breaks (e.g., converting <br> tags or similar HTML line break representations into newline characters \n in the plain text output).

AC4: The system MUST remove common BBCode opening and closing tags (e.g., [b], [/b], [i], [/i], [url=...], [/url]) while preserving the text content enclosed within those tags. (Exception: For link BBCode like [url=...]Text[/url], only the "Text" part is preserved; the URL itself may be lost unless explicitly decided otherwise).

AC5: The system SHOULD handle various forms of whitespace, collapsing redundant whitespace where appropriate without losing significant separation between words or paragraphs.

AC6: The resulting clean text MUST accurately reflect the author's message content and tone (as best as possible without full formatting).

AC7: The system MUST implement robust error handling for malformed HTML or BBCode within a new_text block, logging the issue but attempting to extract as much text as possible.

AC8: The extracted clean text content for each new_text block MUST be available to be included in the final post JSON structure.

AC9: The system MUST log when it encounters an <img> tag or BBCode during cleaning, especially if the handling is uncertain or results in potential data loss (e.g., no alt/title text on an image).

Story 3.5 (Combined): Process Complete Topic and Save Structured JSON File

As the Structured Data Extraction System,

I want to take a specific Topic ID, process all of its archived HTML pages (from Epic 2), extract and assemble the structured data for every post within that topic, and save the complete topic's data into a single JSON file using the defined naming convention,

So that the "Waypoint Archive" contains a structured, machine-readable JSON file for every topic, ready for use in Phase 4.

Acceptance Criteria (ACs):

AC1: The system MUST accept a topic_id as input, and be able to identify and retrieve all archived HTML files corresponding to that topic (across all pages) from the local archive structure (leveraging knowledge from Epic 2's output).

AC2: The system MUST iterate through the HTML files for all pages of the given topic in the correct page order.

AC3: For each HTML page within the topic, the system MUST perform the steps defined in previous stories: read the HTML (Story 3.1 AC1-AC8), identify individual post blocks (Story 3.1 AC4-AC6).

AC4: For each identified post block on every page, the system MUST extract its core metadata (Story 3.2 AC1-AC9).

AC5: For each post block, the system MUST parse its content into structured blocks (new text/quotes) and extract quote details (Combined Story 3.3 AC1-AC10).

AC6: For each new_text block, the system MUST extract the clean text content (Story 3.4 AC1-AC9).

AC7: For each post, the system MUST assemble all the extracted metadata, clean text content, and structured quote data into a single data structure conforming to the agreed-upon JSON schema for a post.

AC8: The system MUST collect all the assembled post data structures for all posts from all pages belonging to the current topic.

AC9: The system MUST structure the collected post data for the topic. This could be an array of post JSON objects at the top level, potentially within a topic-level object that includes topic_id and subforum_id for easy reference within the file itself.

AC10: The system MUST save the final, assembled structured data for the entire topic into a single file.

AC11: The saved file MUST be in valid JSON format.

AC12: The filename MUST adhere to the specified convention: {subforum_id}_{topic_id}.json, and the file should be saved to a user-configurable output directory on the Synology NAS.

AC13: The system MUST implement robust error handling throughout the topic processing (reading pages, parsing posts, extracting data, assembling), logging errors (e.g., failed pages, posts with partial data) but attempting to complete the topic file where possible.

AC14: Upon successful completion for a topic, the system MUST log a confirmation message including the path to the saved JSON file and the number of posts processed for that topic.

AC15: If processing a topic fails irrevocably (e.g., cannot read essential pages), the system MUST log this failure clearly, indicating the problematic topic.

Story 3.6: Orchestrate Full Structured Data Extraction Run with Resumability and Logging

As the Structured Data Extraction System,

I want to read the master list of topics to be processed (derived from Epic 1's index and Epic 2's archive), systematically iterate through these topics, orchestrate the structured data extraction and saving process for each one (using Story 3.5's logic), maintain a persistent state of overall progress, and provide comprehensive logging of the entire run,

So that the full Epic 3 process can be reliably executed, stopped and resumed, monitored, and its outcome (which topics were processed, which failed) is fully traceable.

Acceptance Criteria (ACs):

AC1: The system MUST be configurable with the path to the input list of topics to be processed (e.g., a file generated or used by Epic 2 detailing all successfully archived topics).

AC2: The system MUST successfully read and parse the input list of topics, identifying all unique topic_ids to process.

AC3: The system MUST maintain a persistent state file that records which topic_ids have been successfully processed and saved as JSON files.

AC4: Upon starting, the system MUST check for the existence of the state file. If found, it MUST automatically load the state and resume processing from the next unprocessed topic_id in the input list.

AC5: For each topic_id in the list (skipping those marked as complete in the state file), the system MUST invoke the logic to process that topic and save its JSON file (as defined in Story 3.5).

AC6: If processing a topic (via Story 3.5) is successful, the system MUST update its persistent state to mark that topic_id as completed. The state file MUST be updated frequently (e.g., after each topic, or in batches) to minimize rework on resume.

AC7: If processing a topic (via Story 3.5 AC15) fails irrevocably after internal retries/error handling, the system MUST log this failure clearly (including the topic_id) but continue processing the next topic in the list. These failed topics should ideally be recorded in a separate log or section of the state file for later review.

AC8: The system MUST provide comprehensive operational logging for the entire run, detailing:

Start time and configured parameters.

Confirmation of resuming from a state file, if applicable, and the point of resumption.

The total number of topics identified for processing.

Start and end of processing for each topic.

Success confirmation for each topic, including the saved filename.

Clear logging of any topics skipped (due to prior completion or failure).

Logging of progress statistics (e.g., "Processed X of Y topics", estimated time remaining for the overall run - similar to Epic 2's ETC, though maybe simpler here).

Clear logging of any errors and warnings propagated from Story 3.5.

A final summary upon completion (total processed, total failed).

AC9: The system MUST be designed to be stoppable (e.g., via OS signals like SIGTERM if running in a container) and resume cleanly from the last saved state upon restart.

AC10: The state file format and saving mechanism MUST be robust to prevent corruption during abrupt interruptions.

Story 3.7: End-to-End Testing and Validation of Structured Data Extraction

As the Operator of the Structured Data Extraction System,

I want to execute the complete Epic 3 processing pipeline (orchestrated by Story 3.6, leveraging 3.1-3.5) on a representative sample of the archived data,

So that I can verify that the system reliably extracts and saves the structured data according to all defined requirements and the agreed-upon JSON schema before processing the full archive.

Acceptance Criteria (ACs):

AC1: The complete Epic 3 pipeline, as defined by Stories 3.1 through 3.6, is executable in an end-to-end fashion.

AC2: A representative sample dataset of raw HTML (covering different subforums, topic lengths, posts with/without quotes, varied HTML structures, and potentially some edge cases/errors observed during Epic 2) is available for testing.

AC3: The system can be configured to process only this sample dataset for a test run.

AC4: The system successfully processes the sample dataset without unhandled crashes.

AC5: For a selection of processed topics/posts from the sample output, the generated JSON files MUST be manually or programmatically inspected to verify:

Compliance with the agreed-upon JSON schema.

Accurate extraction of core metadata (post_id, topic_id, subforum_id, page_number, post_order_on_page, post_url, author_username, timestamp).

Correct segmentation of content into new_text and quote blocks.

Accurate extraction of quoted_user, quoted_timestamp, and quoted_text from quote blocks.

Correct and clean extraction of text from new_text blocks, including appropriate emoji representation and BBCode stripping.

Correct ordering of content_blocks.

AC6: The generated JSON filenames MUST adhere to the {subforum_id}_{topic_id}.json convention and be saved in the correct output directory.

AC7: The logging output from the test run (as per Story 3.6 AC8) MUST be reviewed to ensure it accurately reflects the processing steps, any errors encountered within the sample, and overall progress.

AC8: Resumability (Story 3.6 AC3-AC6) MUST be tested by interrupting a test run and verifying that it correctly resumes from the last saved state and completes the processing of the sample.

AC9: Error handling (Story 3.6 AC7 and individual story ACs) within the sample run must be reviewed to ensure errors are logged and handled gracefully without halting the entire process.