package groundtruth

// GroundTruth is the top-level structure matching ground_truth.yaml.
type GroundTruth struct {
	SchemaVersion int     `yaml:"schema_version"`
	GuideURL      string  `yaml:"guide_url"`
	GuideVersion  string  `yaml:"guide_version"`
	Entries       []Entry `yaml:"entries"`
}

// Entry represents a single API migration mapping.
type Entry struct {
	OldAPI       string `yaml:"old_api"`
	NewAPI       string `yaml:"new_api"`
	ActionType   string `yaml:"action_type"`
	Severity     string `yaml:"severity"`
	GuideSection string `yaml:"guide_section"`
	SourceQuote  string `yaml:"source_quote"`
	ReviewedBy   string `yaml:"reviewed_by"`
	ReviewedDate string `yaml:"reviewed_date"`
}
