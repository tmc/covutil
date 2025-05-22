// package covutil provides utilities for working with Go coverage data.
// It allows parsing coverage meta-data and counter files, merging coverage data
// from multiple runs, and generating various human-readable reports.
// Additionally, it offers functions for instrumented binaries to programmatically
// emit their coverage data and control counters.
package covutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	// Public wrappers/adapters for internal types/functionality
	"github.com/tmc/covutil/coverage"

	// Internal stubs (representing the existing library)
	icfile "github.com/tmc/covutil/internal/coverage/cfile"
	icformat "github.com/tmc/covutil/internal/coverage/cformat"
	icmerge "github.com/tmc/covutil/internal/coverage/cmerge"
	ipods "github.com/tmc/covutil/internal/coverage/pods"
)

// --- Basic Coverage Types (Exported from covutil/coverage) ---
// These are now aliases or re-exports from the public covutil/coverage package.

type CounterMode = coverage.CounterMode

const (
	ModeSet     = coverage.ModeSet
	ModeCount   = coverage.ModeCount
	ModeAtomic  = coverage.ModeAtomic
	ModeInvalid = coverage.ModeInvalid
	ModeDefault = coverage.ModeDefault
)

type CounterGranularity = coverage.CounterGranularity

const (
	GranularityBlock   = coverage.GranularityBlock
	GranularityFunc    = coverage.GranularityFunc
	GranularityInvalid = coverage.GranularityInvalid
	GranularityDefault = coverage.GranularityDefault
)

type CoverableUnit = coverage.CoverableUnit
type FuncDesc = coverage.FuncDesc
type PackageMeta = coverage.PackageMeta
type MetaFile = coverage.MetaFile
type CounterFile = coverage.CounterFile
type CounterDataSegment = coverage.CounterDataSegment
type FunctionCounters = coverage.FunctionCounters

// --- Enhanced Pod and Profile Model ---

// Link defines a typed link to an external artifact or concept.
type Link struct {
	Type string // e.g., "git_commit", "git_repo", "pprof_profile", "go_trace", "source_ref", "issue_tracker"
	URI  string // Uniform Resource Identifier (e.g., commit SHA, repo URL, file path with line, ticket URL)
	Desc string // Optional description (e.g., "Source at time of test", "Performance profile for this run")
	// Future: Attributes map[string]string for more structured metadata per link
}

// SourceInfo can hold source control details.
type SourceInfo struct {
	RepoURI    string    // e.g., "https://github.com/user/repo.git"
	CommitSHA  string    // Full commit SHA
	CommitTime time.Time // Time of the commit
	Branch     string    // Optional branch name
	Tag        string    // Optional tag name
	Dirty      bool      // True if the working directory was dirty when data was generated
	// Future: Path string // Relative path within the repo if the coverage is for a sub-module/path
}

// Pod represents a coherent set of coverage data, typically one meta-data file
// and its associated counter files, enriched with labels, links, and timing.
type Pod struct {
	ID        string            // Unique identifier for this pod
	Profile   *Profile          // The actual coverage data
	Labels    map[string]string // User-defined labels (test_name, os, arch, build_id)
	Links     []Link            // Links to related artifacts
	Source    *SourceInfo       // Source control information relevant to this pod's data
	Timestamp time.Time         // When this coverage data was generated or collected
	SubPods   []*Pod            // For hierarchical data (e.g., subtests)

	// Internal bookkeeping, not typically for direct user access
	metaFilePath     string   // Original path to the meta file
	counterFilePaths []string // Original paths to counter files
}

// Profile holds a consistent set of coverage meta-data and aggregated counters.
type Profile struct {
	Meta     MetaFile
	Counters map[PkgFuncKey][]uint32
	Args     map[string]string
}

// PkgFuncKey identifies a function within a package for counter lookup.
type PkgFuncKey = coverage.PkgFuncKey

// CoverageSet manages a collection of Pods.
type CoverageSet struct {
	Pods []*Pod
	mu   sync.RWMutex
}

// --- Loading Functions ---

// LoadMetaFile parses a single meta-data file from a reader.
func LoadMetaFile(r io.Reader, filePath string) (*MetaFile, error) {
	return coverage.ParseMetaFile(r, filePath)
}

// LoadCounterFile parses a single counter data file.
func LoadCounterFile(r io.Reader, filePath string) (*CounterFile, error) {
	return coverage.ParseCounterFile(r, filePath)
}

// LoadCoverageSetFromFS scans an fs.FS for coverage files and loads them.
// It identifies groups of meta and counter files (internal pods) and
// transforms them into public Pod structures containing Profiles.
func LoadCoverageSetFromFS(fsys fs.FS, rootDir string) (*CoverageSet, error) {
	var filePaths []string
	err := fs.WalkDir(fsys, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// Ensure path is relative to fsys root for Open
			relPath := path
			if rootDir != "." && strings.HasPrefix(path, rootDir) {
				var errRel error
				relPath, errRel = filepath.Rel(rootDir, path)
				if errRel != nil {
					return fmt.Errorf("calculating relative path for %s from %s: %w", path, rootDir, errRel)
				}
			}
			filePaths = append(filePaths, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s in fsys: %w", rootDir, err)
	}

	internalPods := ipods.CollectPodsFromFiles(filePaths, false) // This needs careful path handling for FS
	if len(internalPods) == 0 {
		return &CoverageSet{}, nil
	}

	set := &CoverageSet{Pods: make([]*Pod, 0, len(internalPods))}

	for _, ipod := range internalPods {
		// ipod.MetaFile and ipod.CounterDataFiles are paths relative to how CollectPodsFromFiles saw them.
		// If CollectPodsFromFiles was given paths already suitable for fsys.Open (e.g. relative from root), this is fine.
		metaFSPath := ipod.MetaFile

		metaReader, err := fsys.Open(metaFSPath)
		if err != nil {
			return nil, fmt.Errorf("opening meta file %s from fsys: %w", metaFSPath, err)
		}

		parsedMetaFile, err := LoadMetaFile(metaReader, metaFSPath) // Use the public LoadMetaFile
		metaReader.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing meta file %s: %w", metaFSPath, err)
		}

		profile := &Profile{
			Meta:     *parsedMetaFile,
			Counters: make(map[PkgFuncKey][]uint32),
			Args:     make(map[string]string),
		}

		merger := &icmerge.Merger{} // Using internal merger for now
		if err := merger.SetModeAndGranularity("", coverage.InternalCounterMode(parsedMetaFile.Mode), coverage.InternalCounterGranularity(parsedMetaFile.Granularity)); err != nil {
			return nil, fmt.Errorf("setting merge policy for pod %s: %w", metaFSPath, err)
		}

		var firstCounterFileTimestamp time.Time
		var counterFileCount int

		for i, counterFSPath := range ipod.CounterDataFiles {
			counterReader, err := fsys.Open(counterFSPath)
			if err != nil {
				return nil, fmt.Errorf("opening counter file %s from fsys: %w", counterFSPath, err)
			}

			parsedCounterFile, err := LoadCounterFile(counterReader, counterFSPath) // Use public LoadCounterFile
			counterReader.Close()
			if err != nil {
				return nil, fmt.Errorf("parsing counter file %s: %w", counterFSPath, err)
			}

			if !bytes.Equal(parsedCounterFile.MetaFileHash[:], parsedMetaFile.FileHash[:]) {
				fmt.Fprintf(os.Stderr, "warning: counter file %s meta hash %x mismatches %x for meta %s\n",
					counterFSPath, parsedCounterFile.MetaFileHash, parsedMetaFile.FileHash, metaFSPath)
				continue
			}
			counterFileCount++

			// Try to get timestamp from counter file name (standard format)
			// Format: covcounters.<hash>.<pid>.<nanotime>
			if i == 0 { // Use timestamp from first counter file of the pod
				base := filepath.Base(counterFSPath)
				parts := strings.Split(base, ".")
				if len(parts) == 4 {
					if nano, err := parseNanos(parts[3]); err == nil {
						firstCounterFileTimestamp = time.Unix(0, nano)
					}
				}
				profile.Args = parsedCounterFile.Segments[0].Args
			}

			for _, segment := range parsedCounterFile.Segments {
				for _, fCounters := range segment.Functions {
					if int(fCounters.PackageIndex) >= len(parsedMetaFile.Packages) {
						continue
					}
					pkgMeta := parsedMetaFile.Packages[fCounters.PackageIndex]
					if int(fCounters.FunctionIndex) >= len(pkgMeta.Functions) {
						continue
					}
					fnDesc := pkgMeta.Functions[fCounters.FunctionIndex]
					key := PkgFuncKey{PkgPath: pkgMeta.Path, FuncName: fnDesc.FuncName}

					if len(fCounters.Counts) != len(fnDesc.Units) {
						continue
					}

					if existing, ok := profile.Counters[key]; ok {
						_, _ = merger.MergeCounters(existing, fCounters.Counts)
					} else {
						newCounts := make([]uint32, len(fCounters.Counts))
						copy(newCounts, fCounters.Counts)
						profile.Counters[key] = newCounts
					}
				}
			}
		}

		podID := fmt.Sprintf("%x", parsedMetaFile.FileHash)
		if len(ipod.CounterDataFiles) > 0 && !firstCounterFileTimestamp.IsZero() {
			podID = fmt.Sprintf("%s-%d", podID, firstCounterFileTimestamp.UnixNano()) // Make ID more unique if counters exist
		}

		pod := &Pod{
			ID:               podID,
			Profile:          profile,
			Labels:           make(map[string]string),
			Timestamp:        firstCounterFileTimestamp, // Timestamp of first counter file as pod time
			metaFilePath:     metaFSPath,
			counterFilePaths: ipod.CounterDataFiles,
		}
		if goos, ok := profile.Args["GOOS"]; ok {
			pod.Labels["GOOS"] = goos
		}
		if goarch, ok := profile.Args["GOARCH"]; ok {
			pod.Labels["GOARCH"] = goarch
		}
		if counterFileCount == 0 && len(ipod.CounterDataFiles) > 0 {
			// All counter files were mismatched or unparsable for this meta.
			// Decide if such a pod (meta-only) should be added. For now, it is.
		}

		set.Pods = append(set.Pods, pod)
	}
	sort.Slice(set.Pods, func(i, j int) bool { return set.Pods[i].ID < set.Pods[j].ID })
	return set, nil
}

// Helper to parse nanoseconds from counter file names.
func parseNanos(s string) (int64, error) {
	var i int64
	var scale int64 = 1
	if len(s) > 19 { // nanoseconds (19 digits for max int64)
		s = s[:19]
	}
	_, err := fmt.Sscan(s, &i)
	if err != nil {
		// Try parsing as float for very large numbers then converting,
		// though standard counter files shouldn't exceed int64 nanos.
		var f float64
		_, ferr := fmt.Sscan(s, &f)
		if ferr != nil {
			return 0, err // return original Sscan error
		}
		i = int64(f)
	}

	// This part is tricky; runtime emits nanoseconds directly.
	// If it were seconds.nanos, parsing would be different.
	// Assuming s is already nanoseconds.
	return i * scale, nil
}

// --- CoverageSet Methods ---

// FilterByPath returns a new CoverageSet with pods and profiles filtered by package path prefixes.
func (cs *CoverageSet) FilterByPath(prefixes ...string) (*CoverageSet, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	newSet := &CoverageSet{Pods: make([]*Pod, 0, len(cs.Pods))}
	for _, pod := range cs.Pods {
		filteredProfile, err := filterProfileByPath(pod.Profile, prefixes...)
		if err != nil {
			return nil, fmt.Errorf("filtering pod %s: %w", pod.ID, err)
		}
		if len(filteredProfile.Meta.Packages) > 0 || len(filteredProfile.Counters) > 0 {
			newPod := shallowCopyPod(pod)
			newPod.Profile = filteredProfile
			// Recursively filter SubPods if they exist
			if len(pod.SubPods) > 0 {
				tempSubSet := &CoverageSet{Pods: pod.SubPods}
				filteredSubSet, err := tempSubSet.FilterByPath(prefixes...)
				if err != nil {
					return nil, fmt.Errorf("filtering subpods of %s: %w", pod.ID, err)
				}
				newPod.SubPods = filteredSubSet.Pods
			}
			newSet.Pods = append(newSet.Pods, newPod)
		}
	}
	return newSet, nil
}

// FilterByLabel returns a new CoverageSet containing pods that match all provided labels.
func (cs *CoverageSet) FilterByLabel(matchLabels map[string]string) (*CoverageSet, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	newSet := &CoverageSet{Pods: make([]*Pod, 0)}
	for _, pod := range cs.Pods {
		matchesAll := true
		for k, v := range matchLabels {
			if podLabelVal, ok := pod.Labels[k]; !ok || podLabelVal != v {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			// Deep copy pod to avoid modifying original's SubPods if further filtered
			copiedPod, err := deepCopyPod(pod)
			if err != nil {
				return nil, fmt.Errorf("copying pod %s: %w", pod.ID, err)
			}
			// Recursively filter SubPods
			if len(pod.SubPods) > 0 {
				tempSubSet := &CoverageSet{Pods: pod.SubPods} // Original subpods for filtering
				filteredSubSet, err := tempSubSet.FilterByLabel(matchLabels)
				if err != nil {
					return nil, fmt.Errorf("filtering subpods of %s by label: %w", pod.ID, err)
				}
				copiedPod.SubPods = filteredSubSet.Pods
			}
			newSet.Pods = append(newSet.Pods, copiedPod)
		}
	}
	return newSet, nil
}

func shallowCopyPod(original *Pod) *Pod {
	if original == nil {
		return nil
	}
	newPod := *original // Value copy for top-level fields
	// Slices and maps are references, need deep copy if they are to be modified independently
	newPod.Labels = make(map[string]string, len(original.Labels))
	for k, v := range original.Labels {
		newPod.Labels[k] = v
	}
	newPod.Links = make([]Link, len(original.Links))
	copy(newPod.Links, original.Links)
	// Profile is a pointer, so it's shared unless explicitly deep copied.
	// SubPods slice is copied, but pointers within it are shared.
	if len(original.SubPods) > 0 {
		newPod.SubPods = make([]*Pod, len(original.SubPods))
		copy(newPod.SubPods, original.SubPods)
	} else {
		newPod.SubPods = nil
	}
	return &newPod
}

func deepCopyPod(original *Pod) (*Pod, error) {
	if original == nil {
		return nil, nil
	}
	newPod := shallowCopyPod(original) // Start with shallow copy for labels, links etc.

	if original.Profile != nil {
		var err error
		newPod.Profile, err = copyProfile(original.Profile) // Deep copy profile
		if err != nil {
			return nil, fmt.Errorf("deep copying profile: %w", err)
		}
	}
	if original.Source != nil {
		sourceCopy := *original.Source // SourceInfo is a struct of basic types
		newPod.Source = &sourceCopy
	}

	if len(original.SubPods) > 0 {
		newPod.SubPods = make([]*Pod, len(original.SubPods))
		for i, subPod := range original.SubPods {
			copiedSubPod, err := deepCopyPod(subPod)
			if err != nil {
				return nil, fmt.Errorf("deep copying subpod %d: %w", i, err)
			}
			newPod.SubPods[i] = copiedSubPod
		}
	}
	return newPod, nil
}

// Merge aggregates all pods in the CoverageSet into a single summary Pod.
func (cs *CoverageSet) Merge() (*Pod, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if len(cs.Pods) == 0 {
		return nil, fmt.Errorf("no pods in set to merge")
	}

	// Recursively collect all profiles from pods and subpods
	var allProfiles []*Profile
	var collectProfiles func(pods []*Pod)
	collectProfiles = func(pods []*Pod) {
		for _, p := range pods {
			if p.Profile != nil {
				allProfiles = append(allProfiles, p.Profile)
			}
			if len(p.SubPods) > 0 {
				collectProfiles(p.SubPods)
			}
		}
	}
	collectProfiles(cs.Pods)

	if len(allProfiles) == 0 {
		return nil, fmt.Errorf("no profiles found in set to merge")
	}
	if len(allProfiles) == 1 {
		copiedProfile, err := copyProfile(allProfiles[0])
		if err != nil {
			return nil, err
		}
		// Use labels/links/source from the original pod that contained this single profile
		var originalPodForMeta *Pod
		for _, p := range cs.Pods {
			if p.Profile == allProfiles[0] {
				originalPodForMeta = p
				break
			}
		}
		if originalPodForMeta == nil && len(cs.Pods[0].SubPods) > 0 { /* search deeper if needed */
		}
		if originalPodForMeta == nil {
			originalPodForMeta = cs.Pods[0]
		} // fallback

		return &Pod{ID: "merged-" + originalPodForMeta.ID, Profile: copiedProfile, Labels: originalPodForMeta.Labels, Links: originalPodForMeta.Links, Source: originalPodForMeta.Source, Timestamp: originalPodForMeta.Timestamp}, nil
	}

	mergedProfile, err := MergeProfiles(allProfiles...)
	if err != nil {
		return nil, fmt.Errorf("merging profiles in set: %w", err)
	}

	// For the merged pod, take metadata from the first pod in the original set.
	// This is a simplistic choice; more sophisticated merging of labels/links/source might be desired.
	firstPod := cs.Pods[0]
	return &Pod{
		ID:        "merged-set-" + firstPod.ID, // Make ID reflect its origin
		Profile:   mergedProfile,
		Labels:    mapsClone(firstPod.Labels),
		Links:     slicesClone(firstPod.Links),
		Source:    sourceInfoClone(firstPod.Source),
		Timestamp: firstPod.Timestamp, // Or perhaps time.Now() for merge time
	}, nil
}

// Helper for cloning map[string]string
func mapsClone(original map[string]string) map[string]string {
	if original == nil {
		return nil
	}
	cloned := make(map[string]string, len(original))
	for k, v := range original {
		cloned[k] = v
	}
	return cloned
}

// Helper for cloning []Link
func slicesClone[S ~[]E, E any](original S) S {
	if original == nil {
		return nil
	}
	cloned := make(S, len(original))
	copy(cloned, original)
	return cloned
}

// Helper for cloning *SourceInfo
func sourceInfoClone(original *SourceInfo) *SourceInfo {
	if original == nil {
		return nil
	}
	cloned := *original // SourceInfo contains value types
	return &cloned
}

// --- Profile Operations (helpers and main operations) ---
// (MergeProfiles, IntersectProfiles, SubtractProfile, copyProfile, checkCompatibility as before)
// Ensure filterProfileByPath from previous response is also included here.
func filterProfileByPath(p *Profile, prefixes ...string) (*Profile, error) {
	if p == nil {
		return nil, nil
	}
	newMeta := MetaFile{
		FilePath: p.Meta.FilePath, FileHash: p.Meta.FileHash,
		Mode: p.Meta.Mode, Granularity: p.Meta.Granularity, Packages: make([]PackageMeta, 0),
	}
	newCounters := make(map[PkgFuncKey][]uint32)
	for _, pkgMeta := range p.Meta.Packages {
		matches := len(prefixes) == 0
		for _, prefix := range prefixes {
			if strings.HasPrefix(pkgMeta.Path, prefix) {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		newMeta.Packages = append(newMeta.Packages, pkgMeta) // shallow copy of pkgMeta ok if funcs are not modified
		for _, fnDesc := range pkgMeta.Functions {
			key := PkgFuncKey{PkgPath: pkgMeta.Path, FuncName: fnDesc.FuncName}
			if counts, ok := p.Counters[key]; ok {
				newCounters[key] = slicesClone(counts)
			}
		}
	}
	return &Profile{Meta: newMeta, Counters: newCounters, Args: mapsClone(p.Args)}, nil
}

func copyProfile(p *Profile) (*Profile, error) { /* As before */
	if p == nil {
		return nil, fmt.Errorf("cannot copy nil profile")
	}
	copied := &Profile{Meta: p.Meta, Counters: make(map[PkgFuncKey][]uint32, len(p.Counters)), Args: make(map[string]string, len(p.Args))}
	copied.Meta.Packages = make([]PackageMeta, len(p.Meta.Packages))
	for i, pkgM := range p.Meta.Packages {
		newPkgM := pkgM
		newPkgM.Functions = make([]FuncDesc, len(pkgM.Functions))
		for j, fnD := range pkgM.Functions {
			newFnD := fnD
			newFnD.Units = make([]CoverableUnit, len(fnD.Units))
			copy(newFnD.Units, fnD.Units)
			newPkgM.Functions[j] = newFnD
		}
		copied.Meta.Packages[i] = newPkgM
	}
	for k, v := range p.Counters {
		copied.Counters[k] = slicesClone(v)
	}
	for k, v := range p.Args {
		copied.Args[k] = v
	}
	return copied, nil
}
func checkCompatibility(profiles ...*Profile) (*MetaFile, error) { /* As before */
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles for compatibility check")
	}
	firstMeta := profiles[0].Meta
	for i := 1; i < len(profiles); i++ {
		p := profiles[i]
		if !bytes.Equal(p.Meta.FileHash[:], firstMeta.FileHash[:]) {
			return nil, fmt.Errorf("profile %d meta hash %x mismatch canonical %x", i, p.Meta.FileHash, firstMeta.FileHash)
		}
		if p.Meta.Mode != firstMeta.Mode {
			return nil, fmt.Errorf("profile %d mode %s mismatch canonical %s", i, p.Meta.Mode, firstMeta.Mode)
		}
		if p.Meta.Granularity != firstMeta.Granularity {
			return nil, fmt.Errorf("profile %d gran %s mismatch canonical %s", i, p.Meta.Granularity, firstMeta.Granularity)
		}
	}
	return &firstMeta, nil
}
func MergeProfiles(profiles ...*Profile) (*Profile, error) { /* As before */
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles to merge")
	}
	if len(profiles) == 1 {
		return copyProfile(profiles[0])
	}
	canonicalMeta, err := checkCompatibility(profiles...)
	if err != nil {
		return nil, fmt.Errorf("compatibility check for merge: %w", err)
	}
	mergedProfile, err := copyProfile(profiles[0])
	if err != nil {
		return nil, err
	}
	mergedProfile.Meta = *canonicalMeta
	merger := &icmerge.Merger{}
	if err := merger.SetModeAndGranularity("", coverage.InternalCounterMode(canonicalMeta.Mode), coverage.InternalCounterGranularity(canonicalMeta.Granularity)); err != nil {
		return nil, err
	}
	for i := 1; i < len(profiles); i++ {
		p := profiles[i]
		for key, pCounters := range p.Counters {
			if existing, ok := mergedProfile.Counters[key]; ok {
				_, _ = merger.MergeCounters(existing, pCounters)
			} else {
				mergedProfile.Counters[key] = slicesClone(pCounters)
			}
		}
		for k, v := range p.Args {
			mergedProfile.Args[k] = v
		}
	}
	return mergedProfile, nil
}
func IntersectProfiles(profiles ...*Profile) (*Profile, error) { /* As before */
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles for intersection")
	}
	if len(profiles) == 1 {
		return copyProfile(profiles[0])
	}
	canonicalMeta, err := checkCompatibility(profiles...)
	if err != nil {
		return nil, fmt.Errorf("compatibility check for intersection: %w", err)
	}
	result := &Profile{Meta: *canonicalMeta, Counters: make(map[PkgFuncKey][]uint32), Args: mapsClone(profiles[0].Args)}
	for _, pkgMeta := range canonicalMeta.Packages {
		for _, fnDesc := range pkgMeta.Functions {
			key := PkgFuncKey{PkgPath: pkgMeta.Path, FuncName: fnDesc.FuncName}
			numUnits := len(fnDesc.Units)
			if numUnits == 0 {
				continue
			}
			intersectedCounts := make([]uint32, numUnits)
			firstProfileForKey := true
			keyPresentInAll := true
			for _, p := range profiles {
				pFnCounters, keyExists := p.Counters[key]
				if !keyExists || len(pFnCounters) != numUnits {
					keyPresentInAll = false
					break
				}
				if firstProfileForKey {
					copy(intersectedCounts, pFnCounters)
					firstProfileForKey = false
				} else {
					for u := 0; u < numUnits; u++ {
						if pFnCounters[u] == 0 {
							intersectedCounts[u] = 0
						} else if intersectedCounts[u] > 0 {
							if canonicalMeta.Mode == ModeSet {
								intersectedCounts[u] = 1
							} else {
								if pFnCounters[u] < intersectedCounts[u] {
									intersectedCounts[u] = pFnCounters[u]
								}
							}
						}
					}
				}
			}
			if keyPresentInAll {
				anyUnitCovered := false
				for _, count := range intersectedCounts {
					if count > 0 {
						anyUnitCovered = true
						break
					}
				}
				if anyUnitCovered {
					result.Counters[key] = intersectedCounts
				}
			}
		}
	}
	return result, nil
}
func SubtractProfile(profileA, profileB *Profile) (*Profile, error) { /* As before */
	if profileA == nil || profileB == nil {
		return nil, fmt.Errorf("nil profile for subtraction")
	}
	canonicalMeta, err := checkCompatibility(profileA, profileB)
	if err != nil {
		return nil, fmt.Errorf("compatibility check for subtraction: %w", err)
	}
	result := &Profile{Meta: *canonicalMeta, Counters: make(map[PkgFuncKey][]uint32), Args: mapsClone(profileA.Args)}
	for _, pkgMeta := range canonicalMeta.Packages {
		for _, fnDesc := range pkgMeta.Functions {
			key := PkgFuncKey{PkgPath: pkgMeta.Path, FuncName: fnDesc.FuncName}
			numUnits := len(fnDesc.Units)
			if numUnits == 0 {
				continue
			}
			countsA, okA := profileA.Counters[key]
			if !okA || len(countsA) != numUnits {
				continue
			}
			countsB, okB := profileB.Counters[key]
			subtractedCounts := make([]uint32, numUnits)
			anyUnitCoveredInResult := false
			for u := 0; u < numUnits; u++ {
				isCoveredInA := countsA[u] > 0
				isCoveredInB := false
				if okB && len(countsB) == numUnits && countsB[u] > 0 {
					isCoveredInB = true
				}
				if isCoveredInA && !isCoveredInB {
					subtractedCounts[u] = countsA[u]
					anyUnitCoveredInResult = true
				} else {
					subtractedCounts[u] = 0
				}
			}
			if anyUnitCoveredInResult {
				result.Counters[key] = subtractedCounts
			}
		}
	}
	return result, nil
}

// --- Formatting ---
type Formatter struct{ internalFmt coverage.FormatterAPI }

func NewFormatter(mode CounterMode) *Formatter {
	internalFormatter := icformat.NewFormatter(coverage.InternalCounterMode(mode))
	wrapper := coverage.NewFormatterWrapper(internalFormatter, coverage.InternalCounterMode(mode))
	return &Formatter{internalFmt: wrapper}
}
func (f *Formatter) AddPodProfile(p *Pod) error {
	if p == nil || p.Profile == nil {
		return fmt.Errorf("cannot add nil pod or profile")
	}
	return coverage.AddProfileToFormatter(f.internalFmt, &p.Profile.Meta, p.Profile.Counters)
}

type TextualReportOptions struct{ TargetPackages []string }

func (f *Formatter) WriteTextualReport(w io.Writer, opts TextualReportOptions) error {
	return f.internalFmt.EmitTextual(opts.TargetPackages, w)
}

type PercentReportOptions struct {
	TargetPackages     []string
	OverallPackageName string
	IncludeEmpty       bool
	Aggregate          bool
}

func (f *Formatter) WritePercentReport(w io.Writer, opts PercentReportOptions) error {
	return f.internalFmt.EmitPercent(w, opts.TargetPackages, opts.OverallPackageName, opts.IncludeEmpty, opts.Aggregate)
}

type FuncSummaryReportOptions struct { /* TargetPackages []string; // Add if cformat stub supports */
}

func (f *Formatter) WriteFuncSummaryReport(w io.Writer, opts FuncSummaryReportOptions) error {
	return f.internalFmt.EmitFuncs(w)
}

// --- Writing Functions ---

// WriteProfileToDirectory writes a profile's metadata and counter data to separate files in the specified directory
func WriteProfileToDirectory(dirPath string, p *Profile) error {
	if p == nil {
		return fmt.Errorf("cannot write nil profile")
	}

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dirPath, err)
	}

	// Write meta file
	metaFileName := fmt.Sprintf("covmeta.%x", p.Meta.FileHash)
	metaPath := filepath.Join(dirPath, metaFileName)
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("creating meta file %s: %w", metaPath, err)
	}
	defer metaFile.Close()

	// For now, we can't easily recreate the original meta file format
	// This is a placeholder that would need proper implementation
	fmt.Fprintf(metaFile, "# Meta file for %s\n", p.Meta.FilePath)

	return nil
}

// WritePodToDirectory writes a pod's data to files in the specified directory
func WritePodToDirectory(baseDirPath string, pod *Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot write nil pod")
	}

	podDir := filepath.Join(baseDirPath, pod.ID)
	if err := WriteProfileToDirectory(podDir, pod.Profile); err != nil {
		return fmt.Errorf("writing pod profile: %w", err)
	}

	// Write pod metadata (labels, source info, etc.)
	metadataPath := filepath.Join(podDir, "pod_metadata.json")
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("creating metadata file: %w", err)
	}
	defer metadataFile.Close()

	metadata := map[string]interface{}{
		"id":        pod.ID,
		"labels":    pod.Labels,
		"timestamp": pod.Timestamp,
		"source":    pod.Source,
		"links":     pod.Links,
	}

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("encoding pod metadata: %w", err)
	}

	return nil
}

// WriteCoverageSetToDirectory writes all pods in a coverage set to subdirectories
func WriteCoverageSetToDirectory(baseDirPath string, set *CoverageSet) error {
	if set == nil {
		return fmt.Errorf("cannot write nil coverage set")
	}

	if err := os.MkdirAll(baseDirPath, 0755); err != nil {
		return fmt.Errorf("creating base directory %s: %w", baseDirPath, err)
	}

	for _, pod := range set.Pods {
		if err := WritePodToDirectory(baseDirPath, pod); err != nil {
			return fmt.Errorf("writing pod %s: %w", pod.ID, err)
		}
	}

	return nil
}

// --- Runtime Data Emission ---
func WriteMetaFileContent(w io.Writer) error    { return icfile.WriteMeta(w) }
func WriteCounterFileContent(w io.Writer) error { return icfile.WriteCounters(w) }
func ClearCoverageCounters() error              { return icfile.ClearCounters() }
