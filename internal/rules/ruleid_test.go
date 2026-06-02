package rules

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"Hello World", "hello-world"},
		{"---leading---", "leading"},
		{"a!!b##c", "a-b-c"},
		{"UPPER_CASE", "upper-case"},
		{"already-kebab", "already-kebab"},
		{"dots.and.slashes/here", "dots-and-slashes-here"},
		{"  spaces  everywhere  ", "spaces-everywhere"},
		{"MixedCase123", "mixedcase123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRuleIDPrefix(t *testing.T) {
	tests := []struct {
		concern, changeType, want string
	}{
		{"security", "removal", "security-removal"},
		{"http-client", "import", "http-client-import"},
		{"", "import", "general-import"},
		{"security", "", "security-change"},
		{"", "", "general-change"},
		{"JVM Options", "deprecation", "jvm-options-deprecation"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := RuleIDPrefix(tt.concern, tt.changeType)
			if got != tt.want {
				t.Errorf("RuleIDPrefix(%q, %q) = %q, want %q", tt.concern, tt.changeType, got, tt.want)
			}
		})
	}
}

func TestChangeType(t *testing.T) {
	tests := []struct {
		name                                          string
		locationType, providerType, depName, xpath     string
		want                                          string
	}{
		{"dependency", "", "", "org.example.dep", "", "dependency"},
		{"xpath", "", "", "", "//bean", "xml"},
		{"dep takes precedence over xpath", "", "", "dep", "//x", "dependency"},
		{"IMPORT", "IMPORT", "java", "", "", "import"},
		{"PACKAGE", "PACKAGE", "java", "", "", "import"},
		{"METHOD_CALL", "METHOD_CALL", "java", "", "", "method"},
		{"CONSTRUCTOR_CALL", "CONSTRUCTOR_CALL", "java", "", "", "method"},
		{"ANNOTATION", "ANNOTATION", "java", "", "", "annotation"},
		{"INHERITANCE", "INHERITANCE", "java", "", "", "type"},
		{"IMPLEMENTS_TYPE", "IMPLEMENTS_TYPE", "java", "", "", "type"},
		{"builtin provider", "", "builtin", "", "", "pattern"},
		{"default fallback", "", "java", "", "", "change"},
		{"lowercase import", "import", "java", "", "", "import"},
		{"lowercase method_call", "method_call", "java", "", "", "method"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChangeType(tt.locationType, tt.providerType, tt.depName, tt.xpath)
			if got != tt.want {
				t.Errorf("ChangeType(%q, %q, %q, %q) = %q, want %q",
					tt.locationType, tt.providerType, tt.depName, tt.xpath, got, tt.want)
			}
		})
	}
}
