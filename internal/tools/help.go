package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type helpInput struct {
	Topic string `json:"topic"`
}

type helpOutput struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

// GetHelpHandler returns an MCP tool handler for get_help.
func GetHelpHandler() mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input helpInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if input.Topic == "" {
			input.Topic = "all"
		}

		content, err := getHelpContent(input.Topic)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		out := helpOutput{Topic: input.Topic, Content: content}
		data, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil
	}
}

func getHelpContent(topic string) (string, error) {
	switch topic {
	case "condition_types":
		return helpConditionTypes, nil
	case "locations":
		return helpLocations, nil
	case "labels":
		return helpLabels, nil
	case "categories":
		return helpCategories, nil
	case "rule_format":
		return helpRuleFormat, nil
	case "ruleset_format":
		return helpRulesetFormat, nil
	case "examples":
		return helpExamples, nil
	case "all":
		return helpConditionTypes + "\n\n" + helpLocations + "\n\n" + helpLabels + "\n\n" +
			helpCategories + "\n\n" + helpRuleFormat + "\n\n" + helpRulesetFormat + "\n\n" + helpExamples, nil
	default:
		return "", fmt.Errorf("unknown topic %q. Valid topics: condition_types, locations, labels, categories, rule_format, ruleset_format, examples, all", topic)
	}
}

const helpConditionTypes = `## Supported Condition Types

- java.referenced — Java type/class reference. Fields: pattern (required), location (required).
- java.dependency — Maven dependency. Fields: name or nameRegex (one required), lowerbound, upperbound.
- go.referenced — Go symbol reference. Fields: pattern (required).
- go.dependency — Go module dependency. Fields: name or nameRegex (one required), lowerbound, upperbound.
- nodejs.referenced — Node.js symbol reference. Fields: pattern (required).
- csharp.referenced — C# symbol reference. Fields: pattern (required), location (optional: ALL, METHOD, FIELD, CLASS).
- builtin.filecontent — Regex match in file content. Fields: pattern (required), filePattern (optional).
- builtin.file — File existence by name pattern. Fields: pattern (required).
- builtin.xml — XPath match on XML files. Fields: xpath (required), namespaces, filepaths.
- builtin.json — XPath match on JSON files. Fields: xpath (required), filepaths.
- builtin.hasTags — Check for tags. Fields: tags (string array, required).
- builtin.xmlPublicID — DOCTYPE public ID match. Fields: regex (required), namespaces, filepaths.`

const helpLocations = `## Valid Locations

### java.referenced locations (14):
ANNOTATION, IMPORT, CLASS, METHOD_CALL, CONSTRUCTOR_CALL, FIELD, METHOD,
INHERITANCE, IMPLEMENTS_TYPE, ENUM, RETURN_TYPE, VARIABLE_DECLARATION, TYPE, PACKAGE

### csharp.referenced locations (4):
ALL, METHOD, FIELD, CLASS

### Other condition types:
go.referenced, nodejs.referenced, builtin.* — no location field needed.`

const helpLabels = `## Label Format

Labels follow the format: konveyor.io/<key>=<value>

Common labels:
- konveyor.io/source=<source-technology> (e.g., spring-boot-3, java-ee, karaf)
- konveyor.io/target=<target-technology> (e.g., spring-boot-4, quarkus, jakarta-ee)

Multiple source/target labels can be specified for broader matching.`

const helpCategories = `## Valid Categories

- mandatory — Must be changed for the migration to succeed
- optional — Can be changed but migration works without it
- potential — May need to be changed depending on usage`

const helpRuleFormat = `## Rule YAML Format

- ruleID: unique identifier (e.g., spring-boot-00010)
- description: short description (optional)
- category: mandatory | optional | potential
- effort: 1-10 (migration effort estimate)
- labels: list of konveyor.io/ labels
- message: detailed migration guidance (markdown with Before/After code examples)
- links: list of {title, url} documentation links
- when: condition (one of the condition types)
- tag: list of tags (optional, alternative to message)`

const helpRulesetFormat = `## Ruleset YAML Format

- name: ruleset identifier (e.g., spring-boot-4-migration)
- description: human-readable description
- labels: list of konveyor.io/ labels for the entire ruleset`

const helpExamples = `## Example Rule

` + "```yaml" + `
- ruleID: spring-boot-00010
  description: Replace @RequestMapping with specific annotations
  category: mandatory
  effort: 3
  labels:
    - konveyor.io/source=spring-boot-3
    - konveyor.io/target=spring-boot-4
  message: |
    ## Before
    ` + "```java" + `
    @RequestMapping(value = "/api", method = RequestMethod.GET)
    ` + "```" + `
    ## After
    ` + "```java" + `
    @GetMapping("/api")
    ` + "```" + `
    ## Additional info
    - Use @GetMapping, @PostMapping, etc.
  links:
    - title: Spring Boot 4 Migration Guide
      url: https://spring.io/blog/migration
  when:
    java.referenced:
      pattern: org.springframework.web.bind.annotation.RequestMapping
      location: ANNOTATION
` + "```"
