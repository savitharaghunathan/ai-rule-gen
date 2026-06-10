package rules

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFilepaths_ScalarFromTemplate(t *testing.T) {
	const src = `
xpath: //h:property
filepaths: '{{xmlfiles1.filepaths}}'
namespaces:
  h: http://www.hibernate.org/xsd/hibernate-mapping
`
	var got BuiltinXML
	if err := yaml.Unmarshal([]byte(src), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := Filepaths{"{{xmlfiles1.filepaths}}"}
	if !reflect.DeepEqual(got.Filepaths, want) {
		t.Errorf("got %v, want %v", got.Filepaths, want)
	}
}

func TestFilepaths_SequenceForm(t *testing.T) {
	const src = `
xpath: //x
filepaths:
  - pom.xml
  - build.gradle
`
	var got BuiltinXML
	if err := yaml.Unmarshal([]byte(src), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := Filepaths{"pom.xml", "build.gradle"}
	if !reflect.DeepEqual(got.Filepaths, want) {
		t.Errorf("got %v, want %v", got.Filepaths, want)
	}
}

func TestFilepaths_RoundtripSequence(t *testing.T) {
	r := BuiltinXML{XPath: "//x", Filepaths: Filepaths{"pom.xml"}}
	out, err := yaml.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), "- pom.xml") {
		t.Errorf("expected sequence form:\n%s", out)
	}
}

func TestDependency_NameregexAlias(t *testing.T) {
	const src = `
nameregex: org\.jboss\.spec\.javax\.faces\..*
lowerbound: 0.0.0
`
	var got Dependency
	if err := yaml.Unmarshal([]byte(src), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.NameRegex != `org\.jboss\.spec\.javax\.faces\..*` {
		t.Errorf("nameregex: got %q", got.NameRegex)
	}
	if got.Lowerbound != "0.0.0" {
		t.Errorf("lowerbound: got %q", got.Lowerbound)
	}
}

func TestDependency_NameRegexCanonical(t *testing.T) {
	const src = `
name_regex: org\.example\..*
`
	var got Dependency
	if err := yaml.Unmarshal([]byte(src), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.NameRegex != `org\.example\..*` {
		t.Errorf("name_regex: got %q", got.NameRegex)
	}
}

func TestDependency_RoundtripCanonical(t *testing.T) {
	d := Dependency{NameRegex: `org\.example\..*`, Lowerbound: "1.0.0"}
	out, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), "name_regex:") {
		t.Errorf("expected canonical name_regex:\n%s", out)
	}
}
