package groundtruth

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadGroundTruth reads a ground_truth.yaml file.
func ReadGroundTruth(path string) (*GroundTruth, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var gt GroundTruth
	if err := yaml.Unmarshal(data, &gt); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &gt, nil
}

// WriteGroundTruth writes a GroundTruth to a YAML file.
func WriteGroundTruth(gt *GroundTruth, path string) error {
	data, err := yaml.Marshal(gt)
	if err != nil {
		return fmt.Errorf("marshaling ground truth: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Merge combines new entries with an existing GroundTruth.
// Human-reviewed entries (reviewed_by != "" && reviewed_by != "japicmp")
// are preserved. New entries for old_api values not in the existing set are added.
// Existing japicmp-only entries are updated if a new entry exists for the same old_api.
func Merge(existing *GroundTruth, newEntries []Entry) *GroundTruth {
	result := &GroundTruth{
		SchemaVersion: existing.SchemaVersion,
		GuideURL:      existing.GuideURL,
		GuideVersion:  existing.GuideVersion,
	}

	existingByAPI := make(map[string]int, len(existing.Entries))
	for i, e := range existing.Entries {
		existingByAPI[e.OldAPI] = i
	}

	kept := make([]bool, len(existing.Entries))

	for _, ne := range newEntries {
		if idx, ok := existingByAPI[ne.OldAPI]; ok {
			kept[idx] = true
			ee := existing.Entries[idx]
			if isHumanReviewed(ee) {
				result.Entries = append(result.Entries, ee)
			} else {
				result.Entries = append(result.Entries, ne)
			}
		} else {
			result.Entries = append(result.Entries, ne)
		}
	}

	for i, e := range existing.Entries {
		if !kept[i] {
			result.Entries = append(result.Entries, e)
		}
	}

	return result
}

func isHumanReviewed(e Entry) bool {
	return e.ReviewedBy != "" && e.ReviewedBy != "japicmp"
}
