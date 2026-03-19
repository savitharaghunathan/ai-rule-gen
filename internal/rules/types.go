package rules

// We define our own Rule/Condition types rather than importing engine.Rule from
// analyzer-lsp because:
//   - engine.Rule.When is a Conditional interface (runtime evaluation), not a
//     serialization-friendly struct — unusable for YAML generation.
//   - engine.CustomVariable.Pattern is *regexp.Regexp, not a plain string.
//   - Importing engine/ pulls in heavy transitive deps (logr, gRPC, uri, etc.)
//     for what amounts to two trivial types (Category string, Link struct).
//
// Our types produce byte-compatible YAML output that analyzer-lsp can parse.

// Rule is a single Konveyor analyzer rule, YAML-serializable.
type Rule struct {
	RuleID          string           `yaml:"ruleID"`
	Description     string           `yaml:"description,omitempty"`
	Category        Category         `yaml:"category,omitempty"`
	Effort          int              `yaml:"effort,omitempty"`
	Labels          []string         `yaml:"labels,omitempty"`
	Message         string           `yaml:"message,omitempty"`
	Links           []Link           `yaml:"links,omitempty"`
	When            Condition        `yaml:"when"`
	CustomVariables []CustomVariable `yaml:"customVariables,omitempty"`
	Tag             []string         `yaml:"tag,omitempty"`
}

// Category represents the migration category for a rule.
type Category string

const (
	CategoryMandatory Category = "mandatory"
	CategoryOptional  Category = "optional"
	CategoryPotential Category = "potential"
)

// Link is a reference documentation link.
type Link struct {
	URL   string `yaml:"url"`
	Title string `yaml:"title"`
}

// CustomVariable extracts variables from matched code.
type CustomVariable struct {
	Pattern            string `yaml:"pattern"`
	Name               string `yaml:"name"`
	DefaultValue       string `yaml:"defaultValue,omitempty"`
	NameOfCaptureGroup string `yaml:"nameOfCaptureGroup,omitempty"`
}

// Ruleset is metadata for a collection of rules.
type Ruleset struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// Condition represents the "when" block of a rule. Exactly one field should be set.
type Condition struct {
	// Provider-specific conditions
	JavaReferenced    *JavaReferenced    `yaml:"java.referenced,omitempty"`
	JavaDependency    *Dependency        `yaml:"java.dependency,omitempty"`
	GoReferenced      *GoReferenced      `yaml:"go.referenced,omitempty"`
	GoDependency      *Dependency        `yaml:"go.dependency,omitempty"`
	NodejsReferenced  *NodejsReferenced  `yaml:"nodejs.referenced,omitempty"`
	CSharpReferenced  *CSharpReferenced  `yaml:"csharp.referenced,omitempty"`
	BuiltinFilecontent *BuiltinFilecontent `yaml:"builtin.filecontent,omitempty"`
	BuiltinFile       *BuiltinFile       `yaml:"builtin.file,omitempty"`
	BuiltinXML        *BuiltinXML        `yaml:"builtin.xml,omitempty"`
	BuiltinJSON       *BuiltinJSON       `yaml:"builtin.json,omitempty"`
	BuiltinHasTags    []string           `yaml:"builtin.hasTags,omitempty"`
	BuiltinXMLPublicID *BuiltinXMLPublicID `yaml:"builtin.xmlPublicID,omitempty"`

	// Combinators
	Or  []ConditionEntry `yaml:"or,omitempty"`
	And []ConditionEntry `yaml:"and,omitempty"`

	// Chaining fields (applicable to all conditions)
	From   string `yaml:"from,omitempty"`
	As     string `yaml:"as,omitempty"`
	Ignore *bool  `yaml:"ignore,omitempty"`
	Not    *bool  `yaml:"not,omitempty"`
}

// ConditionEntry is a condition within an or/and combinator, with its own chaining fields.
type ConditionEntry struct {
	Condition `yaml:",inline"`
}

// JavaReferenced matches Java symbol references.
type JavaReferenced struct {
	Pattern   string     `yaml:"pattern"`
	Location  string     `yaml:"location,omitempty"`
	Annotated *Annotated `yaml:"annotated,omitempty"`
	Filepaths []string   `yaml:"filepaths,omitempty"`
}

// Annotated filters by annotation presence.
type Annotated struct {
	Pattern  string             `yaml:"pattern"`
	Elements []AnnotatedElement `yaml:"elements,omitempty"`
}

// AnnotatedElement is a name/value pair for annotation element matching.
type AnnotatedElement struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Dependency matches Maven/Go module dependencies by version range.
type Dependency struct {
	Name       string `yaml:"name"`
	NameRegex  string `yaml:"name_regex,omitempty"`
	Upperbound string `yaml:"upperbound,omitempty"`
	Lowerbound string `yaml:"lowerbound,omitempty"`
}

// GoReferenced matches Go symbol references.
type GoReferenced struct {
	Pattern string `yaml:"pattern"`
}

// NodejsReferenced matches Node.js symbol references.
type NodejsReferenced struct {
	Pattern string `yaml:"pattern"`
}

// CSharpReferenced matches C# symbol references.
type CSharpReferenced struct {
	Pattern  string `yaml:"pattern"`
	Location string `yaml:"location,omitempty"`
}

// BuiltinFilecontent matches regex patterns in file contents.
type BuiltinFilecontent struct {
	Pattern     string   `yaml:"pattern"`
	FilePattern string   `yaml:"filePattern,omitempty"`
	Filepaths   []string `yaml:"filepaths,omitempty"`
}

// BuiltinFile matches file name patterns.
type BuiltinFile struct {
	Pattern string `yaml:"pattern"`
}

// BuiltinXML matches XPath expressions in XML files.
type BuiltinXML struct {
	XPath      string            `yaml:"xpath"`
	Namespaces map[string]string `yaml:"namespaces,omitempty"`
	Filepaths  []string          `yaml:"filepaths,omitempty"`
}

// BuiltinJSON matches XPath expressions in JSON files.
type BuiltinJSON struct {
	XPath     string   `yaml:"xpath"`
	Filepaths []string `yaml:"filepaths,omitempty"`
}

// BuiltinXMLPublicID matches DOCTYPE declarations.
type BuiltinXMLPublicID struct {
	Regex      string            `yaml:"regex"`
	Namespaces map[string]string `yaml:"namespaces,omitempty"`
	Filepaths  []string          `yaml:"filepaths,omitempty"`
}

// Java location constants.
const (
	LocationType                = "TYPE"
	LocationInheritance         = "INHERITANCE"
	LocationMethodCall          = "METHOD_CALL"
	LocationConstructorCall     = "CONSTRUCTOR_CALL"
	LocationAnnotation          = "ANNOTATION"
	LocationImplementsType      = "IMPLEMENTS_TYPE"
	LocationEnum                = "ENUM"
	LocationReturnType          = "RETURN_TYPE"
	LocationImport              = "IMPORT"
	LocationVariableDeclaration = "VARIABLE_DECLARATION"
	LocationPackage             = "PACKAGE"
	LocationField               = "FIELD"
	LocationMethod              = "METHOD"
	LocationClass               = "CLASS"
)

// CSharp location constants.
const (
	CSharpLocationAll    = "ALL"
	CSharpLocationMethod = "METHOD"
	CSharpLocationField  = "FIELD"
	CSharpLocationClass  = "CLASS"
)

// ValidJavaLocations is the set of valid Java location values.
var ValidJavaLocations = map[string]bool{
	LocationType: true, LocationInheritance: true, LocationMethodCall: true,
	LocationConstructorCall: true, LocationAnnotation: true, LocationImplementsType: true,
	LocationEnum: true, LocationReturnType: true, LocationImport: true,
	LocationVariableDeclaration: true, LocationPackage: true, LocationField: true,
	LocationMethod: true, LocationClass: true,
}

// ValidCSharpLocations is the set of valid C# location values.
var ValidCSharpLocations = map[string]bool{
	CSharpLocationAll: true, CSharpLocationMethod: true,
	CSharpLocationField: true, CSharpLocationClass: true,
}
