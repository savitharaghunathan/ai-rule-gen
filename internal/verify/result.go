package verify

type Status string

const (
	StatusVerified Status = "verified"
	StatusNotFound Status = "not_found"
	StatusOffline  Status = "registry_offline"
	StatusSkipped  Status = "skipped"
)

type Result struct {
	PatternIndex int      `json:"pattern_index"`
	SourceFQN    string   `json:"source_fqn,omitempty"`
	Status       Status   `json:"status"`
	Evidence     string   `json:"evidence,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
}

type Summary struct {
	Verified        int      `json:"verified"`
	NotFound        int      `json:"not_found"`
	Skipped         int      `json:"skipped"`
	Offline         int      `json:"offline"`
	NotFoundDetails []Result `json:"not_found_details,omitempty"`
}

func Summarize(results []Result) *Summary {
	s := &Summary{}
	for _, r := range results {
		switch r.Status {
		case StatusVerified:
			s.Verified++
		case StatusNotFound:
			s.NotFound++
			s.NotFoundDetails = append(s.NotFoundDetails, r)
		case StatusSkipped:
			s.Skipped++
		case StatusOffline:
			s.Offline++
		}
	}
	return s
}
