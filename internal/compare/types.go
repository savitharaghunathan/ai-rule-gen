// Package compare diffs two Konveyor rulesets: a per-rule coverage matrix
// based on condition keys, and (optionally) a kantra-analyze diff on the
// same app.
package compare

// Result is the full comparison.
type Result struct {
	NameA      string      `json:"name_a"`
	NameB      string      `json:"name_b"`
	RulesDirA  string      `json:"rules_dir_a"`
	RulesDirB  string      `json:"rules_dir_b"`
	RuleCountA int         `json:"rule_count_a"`
	RuleCountB int         `json:"rule_count_b"`
	Matrix     Matrix      `json:"matrix"`
	KantraDiff *KantraDiff `json:"kantra_diff,omitempty"`
}

// Matrix is symmetric coverage in both directions.
type Matrix struct {
	AInB    []RuleCoverage `json:"a_in_b"`
	BInA    []RuleCoverage `json:"b_in_a"`
	Summary MatrixSummary  `json:"summary"`
}

// RuleCoverage is one rule's status against the other set.
type RuleCoverage struct {
	RuleID      string   `json:"rule_id"`
	Status      string   `json:"status"` // covered | partial | missing
	MatchKeys   []string `json:"match_keys"`
	MatchedBy   []string `json:"matched_by,omitempty"`
	PartialBy   []string `json:"partial_by,omitempty"`
	Description string   `json:"description,omitempty"`
}

// MatrixSummary counts statuses by direction.
type MatrixSummary struct {
	AInBCovered int `json:"a_in_b_covered"`
	AInBPartial int `json:"a_in_b_partial"`
	AInBMissing int `json:"a_in_b_missing"`
	BInACovered int `json:"b_in_a_covered"`
	BInAPartial int `json:"b_in_a_partial"`
	BInAMissing int `json:"b_in_a_missing"`
}

// KantraDiff is the file/rule-level delta between two kantra runs.
type KantraDiff struct {
	AppDir          string        `json:"app_dir"`
	RulesFiredA     int           `json:"rules_fired_a"`
	RulesFiredB     int           `json:"rules_fired_b"`
	IncidentsA      int           `json:"incidents_a"`
	IncidentsB      int           `json:"incidents_b"`
	FilesAOnly      []string      `json:"files_a_only"`
	FilesBOnly      []string      `json:"files_b_only"`
	FilesBoth       []FileFinding `json:"files_both"`
	RulesFiredOnlyA []string      `json:"rules_fired_only_a,omitempty"`
	RulesFiredOnlyB []string      `json:"rules_fired_only_b,omitempty"`
}

// FileFinding lists the rules each side fired on a shared file.
type FileFinding struct {
	File   string   `json:"file"`
	RulesA []string `json:"rules_a"`
	RulesB []string `json:"rules_b"`
}
