package eval

// Config holds the inputs for an eval run.
type Config struct {
	RulesDir  string
	AppDir    string
	OutputDir string
}

// EvalResult is the top-level eval output.
type EvalResult struct {
	RuleCount   int              `json:"rule_count"`
	Quality     QualitySummary   `json:"quality"`
	AppCoverage *AppCoverage     `json:"app_coverage,omitempty"`
	RuleDetails []RuleDetail     `json:"rule_details"`
	Overlaps    []Overlap        `json:"overlaps,omitempty"`
}

// QualitySummary aggregates rule quality checks.
type QualitySummary struct {
	TotalRules       int     `json:"total_rules"`
	HasMessage       int     `json:"has_message"`
	HasLinks         int     `json:"has_links"`
	HasEffort        int     `json:"has_effort"`
	HasBeforeAfter   int     `json:"has_before_after"`
	AvgScore         float64 `json:"avg_score"`
	MaxScore         int     `json:"max_score"`
	GuidanceDepthAvg float64 `json:"guidance_depth_avg"`
}

// RuleDetail holds per-rule eval data.
type RuleDetail struct {
	RuleID        string   `json:"rule_id"`
	Description   string   `json:"description"`
	QualityScore  int      `json:"quality_score"`
	QualityMax    int      `json:"quality_max"`
	HasGuidance   bool     `json:"has_guidance"`
	GuidanceDepth int      `json:"guidance_depth"`
	Missing       []string `json:"missing,omitempty"`
	AppIncidents  int      `json:"app_incidents,omitempty"`
	AppFiles      []string `json:"app_files,omitempty"`
}

// AppCoverage holds kantra analyze results.
type AppCoverage struct {
	TotalRules       int                  `json:"total_rules"`
	RulesFired       int                  `json:"rules_fired"`
	TotalIncidents   int                  `json:"total_incidents"`
	NotFired         []string             `json:"not_fired,omitempty"`
	Unmatched        []UnmatchedRule      `json:"unmatched,omitempty"`
	SpecificityGaps  []SpecificityGap     `json:"specificity_gaps,omitempty"`
	EffectiveTotal   int                  `json:"effective_total"`
	EffectiveFired   int                  `json:"effective_fired"`
	EffectivePct     int                  `json:"effective_pct"`
	Violations       map[string]Violation `json:"violations,omitempty"`
}

// UnmatchedRule holds cross-reference data for a not-fired rule.
type UnmatchedRule struct {
	RuleID   string   `json:"rule_id"`
	Pattern  string   `json:"pattern"`
	InApp    bool     `json:"in_app"`
	AppFiles []string `json:"app_files,omitempty"`
	Reason   string   `json:"reason"`
}

// SpecificityGap represents an import in the app that is only covered by a
// broad PACKAGE-level rule but has no dedicated IMPORT/TYPE-level rule.
type SpecificityGap struct {
	BroadRuleID string   `json:"broad_rule_id"`
	ImportFQN   string   `json:"import_fqn"`
	AppFiles    []string `json:"app_files,omitempty"`
}

// Violation holds per-rule analysis results.
type Violation struct {
	Incidents int      `json:"incidents"`
	Files     []string `json:"files"`
}
