package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestFqnTerms(t *testing.T) {
	tests := []struct {
		name string
		fqn  string
		want []string
	}{
		{
			name: "full java FQN with method",
			fqn:  "org.apache.http.HttpResponse.getStatusLine",
			want: []string{"getStatusLine", "HttpResponse"},
		},
		{
			name: "class only",
			fqn:  "org.apache.http.HttpResponse",
			want: []string{"HttpResponse"},
		},
		{
			name: "package wildcard",
			fqn:  "org.apache.http*",
			want: []string{"http*"},
		},
		{
			name: "short method name",
			fqn:  "getStatusLine",
			want: []string{"getStatusLine"},
		},
		{
			name: "empty string",
			fqn:  "",
			want: nil,
		},
		{
			name: "whitespace only",
			fqn:  "   ",
			want: nil,
		},
		{
			name: "trailing wildcard star",
			fqn:  "org.apache.http.*",
			want: nil,
		},
		{
			name: "two-part FQN with lowercase second-to-last",
			fqn:  "http.client",
			want: []string{"client"},
		},
		{
			name: "go module path",
			fqn:  "github.com/old/package.Function",
			want: []string{"Function"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fqnTerms(tt.fqn)
			if !strSliceEqual(got, tt.want) {
				t.Errorf("fqnTerms(%q) = %v, want %v", tt.fqn, got, tt.want)
			}
		})
	}
}

func TestRegexLiterals(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "simple literal",
			pattern: "javax.servlet",
			want:    []string{"javax", "servlet"},
		},
		{
			name:    "regex with groups",
			pattern: `jdbc:postgresql:.*(user=.+|password=.+)`,
			want:    []string{"jdbc:postgresql:", "user=", "password="},
		},
		{
			name:    "escaped dot as literal",
			pattern: `javax\.servlet\.http`,
			want:    []string{"javax.servlet.http"},
		},
		{
			name:    "short segments dropped",
			pattern: `a.bb.cccc`,
			want:    []string{"cccc"},
		},
		{
			name:    "all metacharacters",
			pattern: `^$.*+?{}[]|()`,
			want:    nil,
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    nil,
		},
		{
			name:    "escaped backslash keeps literal",
			pattern: `path\\to\\file`,
			want:    []string{`path\to\file`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := regexLiterals(tt.pattern)
			if !strSliceEqual(got, tt.want) {
				t.Errorf("regexLiterals(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestXpathTerms(t *testing.T) {
	tests := []struct {
		name  string
		xpath string
		want  []string
	}{
		{
			name:  "simple element path",
			xpath: "/project/dependencies/dependency",
			want:  []string{"project", "dependencies", "dependency"},
		},
		{
			name:  "with namespace prefix",
			xpath: "/beans:beans/beans:bean",
			want:  []string{"beans", "bean"},
		},
		{
			name:  "filters out noise words",
			xpath: "/root[text()='value' and not(@disabled)]",
			want:  []string{"root", "value", "disabled"},
		},
		{
			name:  "short segments dropped",
			xpath: "/a/bb/ccc",
			want:  []string{"ccc"},
		},
		{
			name:  "attribute selector",
			xpath: "/configuration/property[@name='hibernate.dialect']",
			want:  []string{"configuration", "property", "name", "hibernate.dialect"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xpathTerms(tt.xpath)
			if !strSliceEqual(got, tt.want) {
				t.Errorf("xpathTerms(%q) = %v, want %v", tt.xpath, got, tt.want)
			}
		})
	}
}

func TestExtractSearchInfo(t *testing.T) {
	tests := []struct {
		name     string
		cond     rules.Condition
		wantLen  int
		wantPat  string
		wantExts []string
	}{
		{
			name: "java.referenced",
			cond: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.HttpResponse",
					Location: "IMPORT",
				},
			},
			wantLen:  1,
			wantPat:  "org.apache.http.HttpResponse",
			wantExts: []string{".java"},
		},
		{
			name: "go.referenced",
			cond: rules.Condition{
				GoReferenced: &rules.GoReferenced{
					Pattern: "github.com/old/pkg.Function",
				},
			},
			wantLen:  1,
			wantPat:  "github.com/old/pkg.Function",
			wantExts: []string{".go"},
		},
		{
			name: "nodejs.referenced",
			cond: rules.Condition{
				NodejsReferenced: &rules.NodejsReferenced{
					Pattern: "express.Router",
				},
			},
			wantLen:  1,
			wantPat:  "express.Router",
			wantExts: []string{".js", ".ts", ".mjs", ".cjs"},
		},
		{
			name: "csharp.referenced",
			cond: rules.Condition{
				CSharpReferenced: &rules.CSharpReferenced{
					Pattern: "System.Net.Http.HttpClient",
				},
			},
			wantLen:  2,
			wantPat:  "System.Net.Http.HttpClient",
			wantExts: []string{".cs"},
		},
		{
			name: "python.referenced",
			cond: rules.Condition{
				PythonReferenced: &rules.PythonReferenced{
					Pattern: "flask.Flask",
				},
			},
			wantLen:  1,
			wantPat:  "flask.Flask",
			wantExts: []string{".py"},
		},
		{
			name: "builtin.filecontent",
			cond: rules.Condition{
				BuiltinFilecontent: &rules.BuiltinFilecontent{
					Pattern:     `spring\.datasource\.url`,
					FilePattern: ".*\\.properties",
				},
			},
			wantLen:  1,
			wantPat:  `spring\.datasource\.url`,
			wantExts: []string{".properties"},
		},
		{
			name: "builtin.xml",
			cond: rules.Condition{
				BuiltinXML: &rules.BuiltinXML{
					XPath: "/project/dependencies/dependency",
				},
			},
			wantLen:  3,
			wantPat:  "/project/dependencies/dependency",
			wantExts: []string{".xml"},
		},
		{
			name: "java.dependency",
			cond: rules.Condition{
				JavaDependency: &rules.Dependency{
					Name: "org.springframework.boot.spring-boot-starter",
				},
			},
			wantLen:  1,
			wantPat:  "org.springframework.boot.spring-boot-starter",
			wantExts: []string{".xml"},
		},
		{
			name: "or combinator merges terms",
			cond: rules.Condition{
				Or: []rules.ConditionEntry{
					{Condition: rules.Condition{
						JavaReferenced: &rules.JavaReferenced{Pattern: "org.old.ClassA"},
					}},
					{Condition: rules.Condition{
						JavaReferenced: &rules.JavaReferenced{Pattern: "org.old.ClassB"},
					}},
				},
			},
			wantLen:  2,
			wantPat:  "org.old.ClassA",
			wantExts: []string{".java"},
		},
		{
			name: "and combinator merges terms and extensions",
			cond: rules.Condition{
				And: []rules.ConditionEntry{
					{Condition: rules.Condition{
						JavaReferenced: &rules.JavaReferenced{Pattern: "org.old.ClassA"},
					}},
					{Condition: rules.Condition{
						BuiltinFilecontent: &rules.BuiltinFilecontent{
							Pattern:     "some.property",
							FilePattern: ".*\\.properties",
						},
					}},
				},
			},
			wantLen:  3,
			wantPat:  "org.old.ClassA",
			wantExts: []string{".java", ".properties"},
		},
		{
			name:     "empty condition",
			cond:     rules.Condition{},
			wantLen:  0,
			wantPat:  "",
			wantExts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms, pat, exts := extractSearchInfo(tt.cond)
			if len(terms) != tt.wantLen {
				t.Errorf("terms count = %d (%v), want %d", len(terms), terms, tt.wantLen)
			}
			if pat != tt.wantPat {
				t.Errorf("pattern = %q, want %q", pat, tt.wantPat)
			}
			if !strSliceEqual(exts, tt.wantExts) {
				t.Errorf("extensions = %v, want %v", exts, tt.wantExts)
			}
		})
	}
}

func TestCrossRefNotFired(t *testing.T) {
	appDir := t.TempDir()

	// Create a Java file containing HttpResponse
	javaDir := filepath.Join(appDir, "src")
	os.MkdirAll(javaDir, 0o755)
	os.WriteFile(filepath.Join(javaDir, "App.java"), []byte(`
import org.apache.http.HttpResponse;
public class App {
    HttpResponse response;
}
`), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "found-rule",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.HttpResponse",
					Location: "IMPORT",
				},
			},
		},
		{
			RuleID: "missing-rule",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "com.nonexistent.NoSuchClass",
					Location: "IMPORT",
				},
			},
		},
	}
	notFired := []string{"found-rule", "missing-rule"}

	results := CrossRefNotFired(ruleList, notFired, appDir)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	var foundResult, missingResult *UnmatchedRule
	for i := range results {
		switch results[i].RuleID {
		case "found-rule":
			foundResult = &results[i]
		case "missing-rule":
			missingResult = &results[i]
		}
	}

	if foundResult == nil {
		t.Fatal("found-rule not in results")
	}
	if !foundResult.InApp {
		t.Error("found-rule.InApp = false, want true")
	}
	if len(foundResult.AppFiles) == 0 {
		t.Error("found-rule.AppFiles is empty, want at least one file")
	}

	if missingResult == nil {
		t.Fatal("missing-rule not in results")
	}
	if missingResult.InApp {
		t.Error("missing-rule.InApp = true, want false")
	}
}

func TestCrossRefSkipDirs(t *testing.T) {
	appDir := t.TempDir()

	// Put the file inside a skip dir — should not be found
	vendorDir := filepath.Join(appDir, "vendor", "lib")
	os.MkdirAll(vendorDir, 0o755)
	os.WriteFile(filepath.Join(vendorDir, "Dep.java"), []byte(`HttpResponse resp;`), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "vendored-rule",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern: "org.apache.http.HttpResponse",
				},
			},
		},
	}

	results := CrossRefNotFired(ruleList, []string{"vendored-rule"}, appDir)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].InApp {
		t.Error("vendored-rule.InApp = true, want false (file is in vendor/)")
	}
}

func TestSourceExtensionsFromFilePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{"properties", ".*\\.properties", []string{".properties"}},
		{"yaml", ".*\\.yaml", []string{".yaml"}},
		{"empty returns all", "", []string{".java", ".go", ".py", ".js", ".ts", ".cs", ".xml", ".yaml", ".yml", ".properties"}},
		{"no extension returns source defaults", "Makefile", []string{".java", ".go", ".py", ".js", ".ts", ".cs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sourceExtensionsFromFilePattern(tt.pattern)
			if !strSliceEqual(got, tt.want) {
				t.Errorf("sourceExtensionsFromFilePattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMergeExts(t *testing.T) {
	got := mergeExts([]string{".java", ".xml"}, []string{".xml", ".properties"})
	want := []string{".java", ".properties", ".xml"}
	if !strSliceEqual(got, want) {
		t.Errorf("mergeExts = %v, want %v", got, want)
	}
}

func strSliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
