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
		{ID: "t1", SubForumID: "sf1", Title: "T1", URL: "topic_u1"},
		{ID: "t2", SubForumID: "sf2", Title: "T2", URL: "topic_u2"},
		{ID: "t3", SubForumID: "sf1", Title: "T3", URL: "topic_u3"},
	}
	subForumDetails := map[string]SubForumNameAndURL{
		"sf1": {Name: "SubForum One", URL: "http://forum.com/sf1"},
		"sf2": {Name: "SubForum Two", URL: "http://forum.com/sf2"},
	}

	want := []data.SubForum{
		{
			ID: 1, Name: "SubForum One", URL: "http://forum.com/sf1", TopicCount: 2,
			Topics: []data.Topic{
				{ID: "t1", SubForumID: "sf1", Title: "T1", URL: "topic_u1"},
				{ID: "t3", SubForumID: "sf1", Title: "T3", URL: "topic_u3"},
			},
		},
		{
			ID: 2, Name: "SubForum Two", URL: "http://forum.com/sf2", TopicCount: 1,
			Topics: []data.Topic{
				{ID: "t2", SubForumID: "sf2", Title: "T2", URL: "topic_u2"},
			},
		},
	}

	got, err := ProcessTopicsAndSubForums(topics, subForumDetails)
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
		t.Errorf("ProcessTopicsAndSubForums() got = %+v, want %+v", got, want)
	}

	// Test case: SubForumID in topic not in subForumDetails map
	topicsMissingSf := []data.Topic{
		{ID: "t4", SubForumID: "sf_missing", Title: "T4", URL: "topic_u4"},
	}
	wantMissingSf := []data.SubForum{
		{
			ID: 99, Name: "sf_missing", URL: "", TopicCount: 1, // Changed "sf_missing" to an arbitrary int like 99
			Topics: []data.Topic{{ID: "t4", SubForumID: "sf_missing", Title: "T4", URL: "topic_u4"}},
		},
	}
	gotMissingSf, err := ProcessTopicsAndSubForums(topicsMissingSf, make(map[string]SubForumNameAndURL))
	if err != nil {
		t.Fatalf("ProcessTopicsAndSubForums() with missing sf error = %v", err)
	}
	if !reflect.DeepEqual(gotMissingSf, wantMissingSf) {
		t.Errorf("ProcessTopicsAndSubForums() with missing sf got = %+v, want %+v", gotMissingSf, wantMissingSf)
	}
}

func TestSortSubForumsByTopicCount(t *testing.T) {
	subForums := []data.SubForum{
		{ID: 1, Name: "SF1", TopicCount: 100},
		{ID: 2, Name: "SF2", TopicCount: 50},
		{ID: 3, Name: "SF3", TopicCount: 150},
	}
	want := []data.SubForum{
		{ID: 2, Name: "SF2", TopicCount: 50},
		{ID: 1, Name: "SF1", TopicCount: 100},
		{ID: 3, Name: "SF3", TopicCount: 150},
	}

	SortSubForumsByTopicCount(subForums)
	if !reflect.DeepEqual(subForums, want) {
		t.Errorf("SortSubForumsByTopicCount() got = %v, want %v", subForums, want)
	}
}

func TestGenerateMasterTopicList(t *testing.T) {
	sortedSubForums := []data.SubForum{
		{
			ID: 1, TopicCount: 1, Topics: []data.Topic{
				{ID: "t1", SubForumID: "sf1"},
			},
		},
		{
			ID: 2, TopicCount: 2, Topics: []data.Topic{
				{ID: "t2", SubForumID: "sf2"},
				{ID: "t1", SubForumID: "sf2"}, // Duplicate Topic ID t1
			},
		},
		{
			ID: 3, TopicCount: 1, Topics: []data.Topic{
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

	// Setup mock subforum list file with URL column
	sfListContent := "SubForumID,SubForumName,SubForumURL\nsf1,Forum One,http://forum.com/f/sf1\nsf2,Forum Two,http://forum.com/f/sf2\n"
	sfListPath := filepath.Join(tempDir, "subforums.csv")
	if err := os.WriteFile(sfListPath, []byte(sfListContent), 0644); err != nil {
		t.Fatalf("Failed to write test subforum list: %v", err)
	}

	// Setup mock topic index directory and files
	topicIndexDir := filepath.Join(tempDir, "topic_indices")
	if err := os.Mkdir(topicIndexDir, 0755); err != nil {
		t.Fatalf("Failed to create test topic index dir: %v", err)
	}

	const topicHeader = "TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked\n"

	topicIndexSF1Content := topicHeader +
		"t101,SF1 Topic 1,url101,userA,10,100,userB,2023-01-01T00:00:00Z,false,false\n" +
		"t102,SF1 Topic 2,url102,userC,20,200,userD,2023-01-02T00:00:00Z,true,false\n"
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf1.csv"), []byte(topicIndexSF1Content), 0644); err != nil {
		t.Fatalf("Failed to write sf1 topic index: %v", err)
	}

	topicIndexSF2Content := topicHeader +
		"t201,SF2 Topic 1,url201,userE,30,300,userF,2023-01-03T00:00:00Z,false,true\n"
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf2.csv"), []byte(topicIndexSF2Content), 0644); err != nil {
		t.Fatalf("Failed to write sf2 topic index: %v", err)
	}

	// Add a malformed topic index file to test skipping (fewer columns than header)
	malformedTopicIndexContent := "TopicID,Title\nbadtopic,NoURL\n" // This will be skipped by ReadTopicIndexCSV due to insufficient columns
	if err := os.WriteFile(filepath.Join(topicIndexDir, "topic_index_sf3_malformed.csv"), []byte(malformedTopicIndexContent), 0644); err != nil {
		t.Fatalf("Failed to write malformed topic index: %v", err)
	}

	cfg := &config.Config{
		SubForumListFile: sfListPath,
		TopicIndexDir:    topicIndexDir,
	}

	// Expected topics (for MasterTopicList check)
	expectedMasterTopics := []data.Topic{
		{ID: "t201", SubForumID: "sf2", Title: "SF2 Topic 1", URL: "url201", AuthorUsername: "userE", Replies: 30, Views: 300, LastPostUsername: "userF", LastPostTimestampRaw: "2023-01-03T00:00:00Z", IsSticky: false, IsLocked: true},
		{ID: "t101", SubForumID: "sf1", Title: "SF1 Topic 1", URL: "url101", AuthorUsername: "userA", Replies: 10, Views: 100, LastPostUsername: "userB", LastPostTimestampRaw: "2023-01-01T00:00:00Z", IsSticky: false, IsLocked: false},
		{ID: "t102", SubForumID: "sf1", Title: "SF1 Topic 2", URL: "url102", AuthorUsername: "userC", Replies: 20, Views: 200, LastPostUsername: "userD", LastPostTimestampRaw: "2023-01-02T00:00:00Z", IsSticky: true, IsLocked: false},
	}
	wantMasterList := data.MasterTopicList{Topics: expectedMasterTopics}

	// Expected subforums (for []data.SubForum check)
	wantSubForums := []data.SubForum{
		{
			ID: 2, Name: "Forum Two", URL: "http://forum.com/f/sf2", TopicCount: 1,
			Topics: []data.Topic{
				{ID: "t201", SubForumID: "sf2", Title: "SF2 Topic 1", URL: "url201", AuthorUsername: "userE", Replies: 30, Views: 300, LastPostUsername: "userF", LastPostTimestampRaw: "2023-01-03T00:00:00Z", IsSticky: false, IsLocked: true},
			},
		},
		{
			ID: 1, Name: "Forum One", URL: "http://forum.com/f/sf1", TopicCount: 2,
			Topics: []data.Topic{
				{ID: "t101", SubForumID: "sf1", Title: "SF1 Topic 1", URL: "url101", AuthorUsername: "userA", Replies: 10, Views: 100, LastPostUsername: "userB", LastPostTimestampRaw: "2023-01-01T00:00:00Z", IsSticky: false, IsLocked: false},
				{ID: "t102", SubForumID: "sf1", Title: "SF1 Topic 2", URL: "url102", AuthorUsername: "userC", Replies: 20, Views: 200, LastPostUsername: "userD", LastPostTimestampRaw: "2023-01-02T00:00:00Z", IsSticky: true, IsLocked: false},
			},
		},
	}

	gotSubForums, gotMasterList, err := LoadAndProcessTopicIndex(cfg)
	if err != nil {
		t.Fatalf("LoadAndProcessTopicIndex() error = %v", err)
	}

	if !reflect.DeepEqual(gotMasterList, wantMasterList) {
		t.Errorf("LoadAndProcessTopicIndex() gotMasterList = %+v, want %+v", gotMasterList, wantMasterList)
	}

	// Sort topics within each subforum for consistent comparison
	for i := range gotSubForums {
		sort.Slice(gotSubForums[i].Topics, func(k, l int) bool { return gotSubForums[i].Topics[k].ID < gotSubForums[i].Topics[l].ID })
	}
	for i := range wantSubForums {
		sort.Slice(wantSubForums[i].Topics, func(k, l int) bool { return wantSubForums[i].Topics[k].ID < wantSubForums[i].Topics[l].ID })
	}

	if !reflect.DeepEqual(gotSubForums, wantSubForums) {
		t.Errorf("LoadAndProcessTopicIndex() gotSubForums = %+v, want %+v", gotSubForums, wantSubForums)
	}

	// Test case: SubForumListFile not found
	cfgNonExistentSFList := &config.Config{
		SubForumListFile: filepath.Join(tempDir, "nonexistent_sflist.csv"),
		TopicIndexDir:    topicIndexDir,
	}
	_, _, err = LoadAndProcessTopicIndex(cfgNonExistentSFList) // Adjusted for new return signature
	if err == nil {
		t.Errorf("LoadAndProcessTopicIndex() with non-existent subforum list file, wantErr = true, got nil error")
	}

	// Test case: TopicIndexDir not found
	cfgNonExistentTopicDir := &config.Config{
		SubForumListFile: sfListPath,
		TopicIndexDir:    filepath.Join(tempDir, "nonexistent_topicdir"),
	}
	_, _, err = LoadAndProcessTopicIndex(cfgNonExistentTopicDir) // Adjusted for new return signature
	if err == nil {
		t.Errorf("LoadAndProcessTopicIndex() with non-existent topic index dir, wantErr = true, got nil error")
	}
}
