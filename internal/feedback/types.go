package feedback

import "time"

// RunSummary is the extracted data from a single pipeline run.
type RunSummary struct {
	Dir       string
	Timestamp time.Time
	Sources   []string
	Targets   []string
	Language  string

	RulesTotal       int
	TestsPassed      int
	TestsFailed      int
	KantraLimitation int
	PassRate         float64

	Patterns []PatternOutcome
}

// PatternOutcome joins a pattern with its verification and test results.
type PatternOutcome struct {
	SourceFQN    string
	LocationType string
	ProviderType string
	Category     string
	Complexity   string
	Concern      string
	HasArtifact  bool
	IsDependency bool
	IsXPath      bool

	VerifyStatus string // verified, not_found, skipped, ""
	VerifyReason string

	TestStatus string // passed, failed, kantra-limitation, untested, ""
}

// FeedbackReport is the output of the analysis.
type FeedbackReport struct {
	RunsAnalyzed   int
	DateRange      string
	MigrationPaths []string

	Overall        OverallStats
	Verify         VerifyAnalysis
	Tests          TestAnalysis
	Recommendations []Recommendation
}

// OverallStats holds aggregate metrics across all analyzed runs.
type OverallStats struct {
	TotalRuns          int
	AveragePassRate    float64
	AverageRulesPerRun float64
	AverageVerifyRate  float64
	PassRateTrend      string // improving, stable, declining
}

// VerifyAnalysis identifies systematic verification failures.
type VerifyAnalysis struct {
	TotalVerified     int
	TotalNotFound     int
	TotalSkipped      int
	RecurringFailures []RecurringFQN
	ByLocationType    map[string]Rate
}

// RecurringFQN is a source_fqn that fails verification repeatedly.
type RecurringFQN struct {
	SourceFQN   string
	Occurrences int
	FailCount   int
	VerifyCount int
	FailRate    float64
}

// TestAnalysis identifies systematic test failures.
type TestAnalysis struct {
	TotalPassed           int
	TotalFailed           int
	TotalKantraLimitation int
	ByLocationType        map[string]Rate
	ByComplexity          map[string]Rate
}

// Rate holds pass/fail (or verified/not_found) counts for a category.
type Rate struct {
	Good  int
	Bad   int
	Value float64
}

// Recommendation is an actionable finding from the analysis.
type Recommendation struct {
	Severity    string // high, medium, low
	Category    string // prompt, reference_doc, pipeline
	Title       string
	Description string
	Evidence    string
	Action      string
}
