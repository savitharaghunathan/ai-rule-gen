package compare

import (
	"reflect"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

func mustRule(t *testing.T, ruleID, descrip, whenYAML string) RawRule {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(whenYAML), &n); err != nil {
		t.Fatalf("bad when yaml: %v", err)
	}
	if n.Kind != yaml.DocumentNode || len(n.Content) == 0 {
		t.Fatalf("unexpected when shape: %d", n.Kind)
	}
	return RawRule{RuleID: ruleID, Description: descrip, When: n.Content[0]}
}

func TestMatchKeysCollapsesJavaxJakarta(t *testing.T) {
	r := mustRule(t, "test", "", `
or:
  - java.referenced:
      pattern: javax.ejb.Stateless
  - java.referenced:
      pattern: jakarta.ejb.Stateless
`)
	keys := MatchKeys(r)
	if !reflect.DeepEqual(keys, []string{"java:javax.ejb.stateless"}) {
		t.Fatalf("got %v, want one collapsed key", keys)
	}
}

func TestMatchKeysHandlesMultipleConditionKinds(t *testing.T) {
	r := mustRule(t, "test", "", `
and:
  - java.dependency:
      name: org.flywaydb:flyway-core
  - builtin.xml:
      xpath: //m:dependency
  - builtin.filecontent:
      pattern: http://xmlns.jcp.org
`)
	got := MatchKeys(r)
	want := []string{
		"dep:org.flywaydb:flyway-core",
		"fc:http://xmlns.jcp.org",
		"xml://m:dependency",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMatchKeysHandlesNameregexAlias(t *testing.T) {
	r := mustRule(t, "test", "", `
or:
  - java.dependency:
      nameregex: org\.jboss\.spec\.javax\.faces\..*
`)
	got := MatchKeys(r)
	want := []string{`dep:org\.jboss\.spec\.javax\.faces\..*`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMatchKeysIgnoresTemplatedFilepaths(t *testing.T) {
	r := mustRule(t, "test", "", `
builtin.xml:
  filepaths: '{{xmlfiles1.filepaths}}'
  xpath: //h:property
`)
	got := MatchKeys(r)
	want := []string{"xml://h:property"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildMatrixCoversExactMatches(t *testing.T) {
	a := []RawRule{
		mustRule(t, "a-1", "", `java.referenced: { pattern: javax.ejb.Stateless }`),
		mustRule(t, "a-2", "", `java.referenced: { pattern: org.example.NoMatch }`),
	}
	b := []RawRule{
		mustRule(t, "b-1", "", `java.referenced: { pattern: jakarta.ejb.Stateless }`),
	}

	m := BuildMatrix(a, b)
	covById := map[string]RuleCoverage{}
	for _, c := range m.AInB {
		covById[c.RuleID] = c
	}
	if covById["a-1"].Status != "covered" {
		t.Errorf("a-1: %s (matched_by=%v)", covById["a-1"].Status, covById["a-1"].MatchedBy)
	}
	if covById["a-2"].Status != "missing" {
		t.Errorf("a-2: %s", covById["a-2"].Status)
	}
	if m.Summary.AInBCovered != 1 || m.Summary.AInBMissing != 1 {
		t.Errorf("summary: %+v", m.Summary)
	}
}

func TestBuildMatrixMarksPackagePrefixAsPartial(t *testing.T) {
	a := []RawRule{
		mustRule(t, "broad", "", `java.referenced: { pattern: javax.ejb }`),
	}
	b := []RawRule{
		mustRule(t, "narrow", "", `java.referenced: { pattern: javax.ejb.Stateless }`),
	}
	m := BuildMatrix(a, b)
	if m.AInB[0].Status != "partial" {
		t.Errorf("a→b: %s (partial_by=%v)", m.AInB[0].Status, m.AInB[0].PartialBy)
	}
	if m.BInA[0].Status != "partial" {
		t.Errorf("b→a: %s", m.BInA[0].Status)
	}
}
