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
}

// RuleDetail holds per-rule eval data.
type RuleDetail struct {
	RuleID       string       `json:"rule_id"`
	Description  string       `json:"description"`
	QualityScore int          `json:"quality_score"`
	QualityMax   int          `json:"quality_max"`
	HasGuidance  bool         `json:"has_guidance"`
	Missing      []string     `json:"missing,omitempty"`
	AppIncidents int          `json:"app_incidents,omitempty"`
	AppFiles     []string     `json:"app_files,omitempty"`
}

// AppCoverage holds kantra analyze results.
type AppCoverage struct {
	TotalRules       int                  `json:"total_rules"`
	RulesFired       int                  `json:"rules_fired"`
	TotalIncidents   int                  `json:"total_incidents"`
	NotFired         []string             `json:"not_fired,omitempty"`
	Unmatched        []UnmatchedRule      `json:"unmatched,omitempty"`
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

// Violation holds per-rule analysis results.
type Violation struct {
	Incidents int      `json:"incidents"`
	Files     []string `json:"files"`
}
