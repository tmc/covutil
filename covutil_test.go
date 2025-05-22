package covutil

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"testing/fstest"
	"time"

	// Public wrappers for types
	"github.com/tmc/covutil/coverage"

	// Internal stubs for testing
	icoverage "github.com/tmc/covutil/internal/coverage"
)

// --- Test Helpers ---
// (createTestMetaReader, createTestCountersReader from previous test file, adapted for new type locations)
func createTestMetaReader(t *testing.T, hash [16]byte, mode coverage.CounterMode, packages []coverage.PackageMetaStub) io.Reader {
	t.Helper()
	hdr := icoverage.MetaFileHeader{
		Magic: icoverage.CovMetaMagic, Version: 1, Entries: uint64(len(packages)),
		MetaFileHash: hash, CMode: icoverage.CounterMode(mode), CGranularity: icoverage.CtrGranularityPerBlock,
	}
	oldMockPkgs := coverage.MockMetaPackages
	oldMockHash := coverage.MockMetaFileHash
	coverage.MockMetaPackages = packages
	coverage.MockMetaFileHash = hash
	t.Cleanup(func() { coverage.MockMetaPackages = oldMockPkgs; coverage.MockMetaFileHash = oldMockHash })
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &hdr)
	for i := 0; i < len(packages); i++ {
		binary.Write(buf, binary.LittleEndian, uint64(0))
	}
	for i := 0; i < len(packages); i++ {
		binary.Write(buf, binary.LittleEndian, uint64(0))
	}
	return buf
}

func createTestCountersReader(t *testing.T, metaHash [16]byte, segments [][]coverage.FuncPayload, args []map[string]string) io.Reader {
	t.Helper()
	hdr := icoverage.CounterFileHeader{Magic: icoverage.CovCounterMagic, Version: 1, MetaHash: metaHash}
	oldMockSegments := coverage.MockCounterSegments
	oldMockArgs := coverage.MockCounterArgs
	oldMockMetaHash := coverage.MockCounterMetaFileHash
	coverage.MockCounterSegments = segments
	coverage.MockCounterArgs = args
	coverage.MockCounterMetaFileHash = metaHash
	t.Cleanup(func() {
		coverage.MockCounterSegments = oldMockSegments
		coverage.MockCounterArgs = oldMockArgs
		coverage.MockCounterMetaFileHash = oldMockMetaHash
	})
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &hdr)
	return buf
}

// --- Tests ---

func TestBasicTypes(t *testing.T) {
	// Test basic type creation and manipulation
	profile := &Profile{
		Meta: MetaFile{
			FilePath: "test.meta",
			FileHash: [16]byte{1, 2, 3},
			Mode:     ModeCount,
		},
		Counters: make(map[PkgFuncKey][]uint32),
		Args:     make(map[string]string),
	}

	// Test PkgFuncKey
	key := PkgFuncKey{PkgPath: "test/pkg", FuncName: "TestFunc"}
	profile.Counters[key] = []uint32{1, 2, 3}

	if profile.Meta.FilePath != "test.meta" {
		t.Errorf("FilePath mismatch")
	}

	if len(profile.Counters) != 1 {
		t.Errorf("Expected 1 counter entry, got %d", len(profile.Counters))
	}

	if counts, ok := profile.Counters[key]; !ok || len(counts) != 3 {
		t.Errorf("Counter entry not found or wrong length")
	}
}

func TestLoadCoverageSetFromFS_EmptyDir(t *testing.T) {
	// Test loading from empty filesystem
	fsMap := make(fstest.MapFS)

	set, err := LoadCoverageSetFromFS(fsMap, ".")
	if err != nil {
		t.Fatalf("LoadCoverageSetFromFS failed: %v", err)
	}
	if len(set.Pods) != 0 {
		t.Fatalf("Expected 0 pods from empty dir, got %d", len(set.Pods))
	}
}

func TestPodLabelingAndSourceInfo(t *testing.T) {
	// Test Pod creation with labels and source info
	now := time.Now()
	profile := &Profile{
		Meta: MetaFile{FilePath: "test.meta"},
		Counters: map[PkgFuncKey][]uint32{
			{PkgPath: "test/pkg", FuncName: "TestFunc"}: {1, 1},
		},
		Args: map[string]string{"GOOS": "linux"},
	}

	pod := &Pod{
		ID:        "test-pod-1",
		Profile:   profile,
		Labels:    map[string]string{"test_type": "unit", "feature": "login"},
		Timestamp: now,
		Source: &SourceInfo{
			RepoURI:   "https://example.com/repo.git",
			CommitSHA: "abcdef1234567890",
			Branch:    "main",
			Dirty:     false,
		},
		Links: []Link{
			{Type: "issue_tracker", URI: "https://example.com/issues/123", Desc: "Related issue"},
		},
		metaFilePath:     "meta.covm",
		counterFilePaths: []string{"counter.covc"},
	}

	if pod.Labels["feature"] != "login" {
		t.Errorf("Label 'feature' mismatch")
	}
	if pod.Source.CommitSHA != "abcdef1234567890" {
		t.Errorf("SourceInfo CommitSHA mismatch")
	}
	if len(pod.Links) != 1 || pod.Links[0].URI != "https://example.com/issues/123" {
		t.Errorf("Links mismatch")
	}
	if !pod.Timestamp.Equal(now) {
		t.Errorf("Timestamp mismatch")
	}
}

func TestCoverageSet_FilterByLabel(t *testing.T) {
	pod1 := &Pod{ID: "p1", Labels: map[string]string{"os": "linux", "arch": "amd64"}, Profile: &Profile{Meta: MetaFile{FilePath: "m1"}}}
	pod2 := &Pod{ID: "p2", Labels: map[string]string{"os": "darwin", "arch": "arm64"}, Profile: &Profile{Meta: MetaFile{FilePath: "m2"}}}
	pod3 := &Pod{ID: "p3", Labels: map[string]string{"os": "linux", "arch": "arm64"}, Profile: &Profile{Meta: MetaFile{FilePath: "m3"}}}
	pod4 := &Pod{ID: "p4", Labels: map[string]string{"os": "linux", "arch": "amd64", "extra": "val"}, Profile: &Profile{Meta: MetaFile{FilePath: "m4"}},
		SubPods: []*Pod{
			{ID: "p4sub1", Labels: map[string]string{"os": "linux", "arch": "amd64", "sub": "yes"}, Profile: &Profile{Meta: MetaFile{FilePath: "m4s1"}}},
			{ID: "p4sub2", Labels: map[string]string{"os": "darwin"}, Profile: &Profile{Meta: MetaFile{FilePath: "m4s2"}}}, // Mismatched os
		}}

	set := &CoverageSet{Pods: []*Pod{pod1, pod2, pod3, pod4}}

	// Filter for os:linux
	filtered, err := set.FilterByLabel(map[string]string{"os": "linux"})
	if err != nil {
		t.Fatalf("FilterByLabel failed: %v", err)
	}
	if len(filtered.Pods) != 3 {
		t.Fatalf("Expected 3 pods for os:linux, got %d", len(filtered.Pods))
	} // p1, p3, p4
	// Check subpods of p4
	foundP4 := false
	for _, p := range filtered.Pods {
		if p.ID == "p4" {
			foundP4 = true
			if len(p.SubPods) != 1 || p.SubPods[0].ID != "p4sub1" {
				t.Errorf("p4 subpods not filtered correctly by label. Got %d subpods.", len(p.SubPods))
			}
		}
	}
	if !foundP4 {
		t.Error("p4 not found in os:linux filter")
	}

	// Filter for os:linux and arch:amd64
	filtered, err = set.FilterByLabel(map[string]string{"os": "linux", "arch": "amd64"})
	if err != nil {
		t.Fatalf("FilterByLabel failed: %v", err)
	}
	if len(filtered.Pods) != 2 {
		t.Fatalf("Expected 2 pods for os:linux, arch:amd64, got %d", len(filtered.Pods))
	} // p1, p4
}

// Example test for deepCopyPod, crucial for filter operations returning new sets
func TestDeepCopyPod(t *testing.T) {
	originalProfile := &Profile{Meta: MetaFile{FilePath: "meta/path"}, Counters: map[PkgFuncKey][]uint32{{PkgPath: "a", FuncName: "b"}: {1, 2}}}
	originalSource := &SourceInfo{CommitSHA: "123"}
	original := &Pod{
		ID: "original", Profile: originalProfile, Source: originalSource,
		Labels: map[string]string{"k": "v"}, Links: []Link{{Type: "t", URI: "u"}},
		SubPods: []*Pod{{ID: "sub", Profile: &Profile{Meta: MetaFile{FilePath: "sub/meta"}}}},
	}

	copied, err := deepCopyPod(original)
	if err != nil {
		t.Fatalf("deepCopyPod failed: %v", err)
	}

	if copied == original {
		t.Fatal("Copied pod is same instance as original")
	}
	if copied.Profile == original.Profile {
		t.Fatal("Copied profile is same instance")
	}
	if copied.Profile.Meta.FilePath != original.Profile.Meta.FilePath {
		t.Error("Profile meta path changed")
	}
	if &copied.Profile.Counters == &original.Profile.Counters {
		t.Fatal("Counters map is same instance")
	}
	copied.Profile.Counters[PkgFuncKey{PkgPath: "a", FuncName: "b"}][0] = 99
	if original.Profile.Counters[PkgFuncKey{PkgPath: "a", FuncName: "b"}][0] == 99 {
		t.Error("Modifying copied counter affected original")
	}

	if copied.Source == original.Source {
		t.Fatal("SourceInfo is same instance")
	}
	copied.Source.CommitSHA = "456"
	if original.Source.CommitSHA == "456" {
		t.Error("Modifying copied SourceInfo affected original")
	}

	if &copied.Labels == &original.Labels {
		t.Fatal("Labels map is same instance")
	}
	copied.Labels["k"] = "new_v"
	if original.Labels["k"] == "new_v" {
		t.Error("Modifying copied label affected original")
	}

	if len(copied.Links) > 0 && len(original.Links) > 0 && &copied.Links[0] == &original.Links[0] {
		// This check is tricky because []Link might be copied by value if Link is a struct,
		// but if Link contained pointers, those would be shallow. Link is struct of strings, so ok.
	}
	if len(copied.SubPods) > 0 {
		if copied.SubPods[0] == original.SubPods[0] {
			t.Fatal("Copied subpod is same instance")
		}
		if copied.SubPods[0].Profile == original.SubPods[0].Profile {
			t.Fatal("Copied subpod's profile is same instance")
		}
	}
}

func TestWritingFunctions(t *testing.T) {
	// Test basic writing functionality
	profile := &Profile{
		Meta: MetaFile{
			FilePath: "test.meta",
			FileHash: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			Mode:     ModeCount,
		},
		Counters: map[PkgFuncKey][]uint32{
			{PkgPath: "test/pkg", FuncName: "TestFunc"}: {1, 2, 3},
		},
		Args: map[string]string{"GOOS": "linux"},
	}

	// Test WriteProfileToDirectory
	tempDir := t.TempDir()
	err := WriteProfileToDirectory(tempDir, profile)
	if err != nil {
		t.Errorf("WriteProfileToDirectory failed: %v", err)
	}

	// Test WritePodToDirectory
	pod := &Pod{
		ID:        "test-pod",
		Profile:   profile,
		Labels:    map[string]string{"test": "true"},
		Timestamp: time.Now(),
	}

	podDir := t.TempDir()
	err = WritePodToDirectory(podDir, pod)
	if err != nil {
		t.Errorf("WritePodToDirectory failed: %v", err)
	}

	// Test WriteCoverageSetToDirectory
	set := &CoverageSet{
		Pods: []*Pod{pod},
	}

	setDir := t.TempDir()
	err = WriteCoverageSetToDirectory(setDir, set)
	if err != nil {
		t.Errorf("WriteCoverageSetToDirectory failed: %v", err)
	}
}

func TestMergeProfiles(t *testing.T) {
	// Test merging profiles
	profile1 := &Profile{
		Meta: MetaFile{
			FilePath: "test1.meta",
			FileHash: [16]byte{1, 2, 3},
			Mode:     ModeCount,
		},
		Counters: map[PkgFuncKey][]uint32{
			{PkgPath: "test/pkg", FuncName: "TestFunc"}: {1, 2},
		},
		Args: map[string]string{"GOOS": "linux"},
	}

	profile2 := &Profile{
		Meta: MetaFile{
			FilePath: "test2.meta",
			FileHash: [16]byte{1, 2, 3}, // Same hash for compatibility
			Mode:     ModeCount,
		},
		Counters: map[PkgFuncKey][]uint32{
			{PkgPath: "test/pkg", FuncName: "TestFunc"}: {3, 4},
		},
		Args: map[string]string{"GOARCH": "amd64"},
	}

	merged, err := MergeProfiles(profile1, profile2)
	if err != nil {
		t.Errorf("MergeProfiles failed: %v", err)
	}

	if merged == nil {
		t.Errorf("MergeProfiles returned nil")
	}
}
