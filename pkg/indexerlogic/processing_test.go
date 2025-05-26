package indexerlogic

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data"
)

func TestProcessTopicsAndSubForums(t *testing.T) {
	topics := []data.Topic{
		{ID: "t1", SubForumID: "sf1", Title: "T1", URL: "u1"},
		{ID: "t2", SubForumID: "sf2", Title: "T2", URL: "u2"},
		{ID: "t3", SubForumID: "sf1", Title: "T3", URL: "u3"},
	}
	subForumNames := map[string]string{
		"sf1": "SubForum One",
		"sf2": "SubForum Two",
	}

	want := []data.SubForum{
		{
			ID: "sf1", Name: "SubForum One", TopicCount: 2,
			Topics: []data.Topic{
				{ID: "t1", SubForumID: "sf1", Title: "T1", URL: "u1"},
				{ID: "t3", SubForumID: "sf1", Title: "T3", URL: "u3"},
			},
		},
		{
			ID: "sf2", Name: "SubForum Two", TopicCount: 1,
			Topics: []data.Topic{
				{ID: "t2", SubForumID: "sf2", Title: "T2", URL: "u2"},
			},
		},
	}

	got, err := ProcessTopicsAndSubForums(topics, subForumNames)
	if err != nil {
		t.Fatalf("ProcessTopicsAndSubForums() error = %v", err)
	}

	// Sort slices for consistent comparison, as map iteration order is not guaranteed
	sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
	sort.Slice(want, func(i, j int) bool { return want[i].ID < want[j].ID })
	for i := range got {
		sort.Slice(got[i].Topics, func(k, l int) bool { return got[i].Topics[k].ID < got[i].Topics[l].ID })
		sort.Slice(want[i].Topics, func(k, l int) bool { return want[i].Topics[k].ID < want[i].Topics[l].ID })
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ProcessTopicsAndSubForums() got = %v, want %v", got, want)
	}

	// Test case: SubForumID in topic not in subForumNames map
	topicsMissingSf := []data.Topic{
		{ID: "t4", SubForumID: "sf_missing", Title: "T4", URL: "u4"},
	}
	wantMissingSf := []data.SubForum{
		{
			ID: "sf_missing", Name: "sf_missing", TopicCount: 1, // Expects ID as name
			Topics: []data.Topic{{ID: "t4", SubForumID: "sf_missing", Title: "T4", URL: "u4"}},
		},
	}
	gotMissingSf, err := ProcessTopicsAndSubForums(topicsMissingSf, subForumNames)
	if err != nil {
		t.Fatalf("ProcessTopicsAndSubForums() with missing sf error = %v", err)
	}
	if !reflect.DeepEqual(gotMissingSf, wantMissingSf) {
		t.Errorf("ProcessTopicsAndSubForums() with missing sf got = %v, want %v", gotMissingSf, wantMissingSf)
	}
}

func TestSortSubForumsByTopicCount(t *testing.T) {
	subForums := []data.SubForum{
		{ID: "sf1", Name: "SF1", TopicCount: 100},
		{ID: "sf2", Name: "SF2", TopicCount: 50},
		{ID: "sf3", Name: "SF3", TopicCount: 150},
	}
	want := []data.SubForum{
		{ID: "sf2", Name: "SF2", TopicCount: 50},
		{ID: "sf1", Name: "SF1", TopicCount: 100},
		{ID: "sf3", Name: "SF3", TopicCount: 150},
	}

	SortSubForumsByTopicCount(subForums)
	if !reflect.DeepEqual(subForums, want) {
		t.Errorf("SortSubForumsByTopicCount() got = %v, want %v", subForums, want)
	}
}

func TestGenerateMasterTopicList(t *testing.T) {
	sortedSubForums := []data.SubForum{
		{
			ID: "sf1", TopicCount: 1, Topics: []data.Topic{
				{ID: "t1", SubForumID: "sf1"},
			},
		},
		{
			ID: "sf2", TopicCount: 2, Topics: []data.Topic{
				{ID: "t2", SubForumID: "sf2"},
				{ID: "t1", SubForumID: "sf2"}, // Duplicate Topic ID t1
			},
		},
		{
			ID: "sf3", TopicCount: 1, Topics: []data.Topic{
				{ID: "t3", SubForumID: "sf3"},
			},
		},
	}

	want := data.MasterTopicList{
		Topics: []data.Topic{
			{ID: "t1", SubForumID: "sf1"}, // First occurrence of t1
			{ID: "t2", SubForumID: "sf2"},
			{ID: "t3", SubForumID: "sf3"},
		},
	}

	got := GenerateMasterTopicList(sortedSubForums)

	// Need to sort got.Topics if order within subforums isn't guaranteed by GenerateMasterTopicList
	// and the test data sf.Topics are not pre-sorted in the exact way GML outputs them.
	// The de-duplication happens based on first encountered. So order should be predictable.

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GenerateMasterTopicList() got = %v, want %v", got, want)
	}

	// Test with empty input
	emptySortedSubForums := []data.SubForum{}
	wantEmpty := data.MasterTopicList{Topics: []data.Topic{}}
	gotEmpty := GenerateMasterTopicList(emptySortedSubForums)
	if !reflect.DeepEqual(gotEmpty, wantEmpty) {
		t.Errorf("GenerateMasterTopicList() with empty input got = %v, want %v", gotEmpty, wantEmpty)
	}
}

// TestLoadAndProcessTopicIndex is a more integration-style test.
// It sets up mock files and checks the overall outcome.
func TestLoadAndProcessTopicIndex(t *testing.T) {
	tempDir := t.TempDir()

	// Setup mock subforum list file
	sfListContent := `SubForumID,SubForumName
sf1,Forum One
sf2,Forum Two
`
	sfListPath := filepath.Join(tempDir, "subforums.csv")
	if err := os.WriteFile(sfListPath, []byte(sfListContent), 0644); err != nil {
		t.Fatalf("Failed to write test subforum list: %v", err)
	}

	// Setup mock topic index directory and files
	topicIndexDir := filepath.Join(tempDir, "topic_indices")
	if err := os.Mkdir(topicIndexDir, 0755); err != nil {
		t.Fatalf("Failed to create test topic index dir: %v", err)
	}

	topicIndexSF1Content := `TopicID,Title,URL
t101,SF1 Topic 1,url101
t102,SF1 Topic 2,url102
`
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf1.csv"), []byte(topicIndexSF1Content), 0644); err != nil {
		t.Fatalf("Failed to write sf1 topic index: %v", err)
	}

	topicIndexSF2Content := `TopicID,Title,URL
t201,SF2 Topic 1,url201
`
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf2.csv"), []byte(topicIndexSF2Content), 0644); err != nil {
		t.Fatalf("Failed to write sf2 topic index: %v", err)
	}
	// Add a malformed topic index file to test skipping
	malformedTopicIndexContent := `TopicID,Title
badtopic,NoURL
`
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf3_malformed.csv"), []byte(malformedTopicIndexContent), 0644); err != nil {
		t.Fatalf("Failed to write malformed topic index: %v", err)
	}

	cfg := &config.Config{
		SubForumListFile: sfListPath,
		TopicIndexDir:    topicIndexDir,
	}

	wantTopics := []data.Topic{
		// Expected order is sf2 (1 topic) then sf1 (2 topics)
		{ID: "t201", SubForumID: "sf2", Title: "SF2 Topic 1", URL: "url201"},
		{ID: "t101", SubForumID: "sf1", Title: "SF1 Topic 1", URL: "url101"},
		{ID: "t102", SubForumID: "sf1", Title: "SF1 Topic 2", URL: "url102"},
	}
	wantMasterList := data.MasterTopicList{Topics: wantTopics}

	gotMasterList, err := LoadAndProcessTopicIndex(cfg)
	if err != nil {
		t.Fatalf("LoadAndProcessTopicIndex() error = %v", err)
	}

	if !reflect.DeepEqual(gotMasterList, wantMasterList) {
		t.Errorf("LoadAndProcessTopicIndex() got = %v, want %v", gotMasterList, wantMasterList)
	}

	// Test case: SubForumListFile not found
	cfgNonExistentSFList := &config.Config{
		SubForumListFile: filepath.Join(tempDir, "nonexistent_sflist.csv"),
		TopicIndexDir:    topicIndexDir,
	}
	_, err = LoadAndProcessTopicIndex(cfgNonExistentSFList)
	if err == nil {
		t.Errorf("LoadAndProcessTopicIndex() with non-existent subforum list file, wantErr = true, got nil error")
	}

	// Test case: TopicIndexDir not found
	cfgNonExistentTopicDir := &config.Config{
		SubForumListFile: sfListPath,
		TopicIndexDir:    filepath.Join(tempDir, "nonexistent_topicdir"),
	}
	_, err = LoadAndProcessTopicIndex(cfgNonExistentTopicDir)
	if err == nil {
		t.Errorf("LoadAndProcessTopicIndex() with non-existent topic index dir, wantErr = true, got nil error")
	}
}
