package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// Field describes one input/output item in a skill contract.
type Field struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Required  bool     `json:"required"`
	Enum      []string `json:"enum,omitempty"`
	ItemsType string   `json:"items_type,omitempty"`
}

// SkillContract is the machine-readable contract for a skill.
type SkillContract struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Inputs  []Field `json:"inputs"`
	Returns []Field `json:"returns"`
}

// ValidationError represents a single contract validation issue.
type ValidationError struct {
	Field   string
	Message string
}

// Load reads and validates a skill contract JSON file.
func Load(path string) (*SkillContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading contract %s: %w", path, err)
	}

	var c SkillContract
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing contract %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid contract %s: %w", path, err)
	}
	return &c, nil
}

// Validate checks contract structure consistency.
func (c *SkillContract) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if err := validateFields("inputs", c.Inputs); err != nil {
		return err
	}
	if err := validateFields("returns", c.Returns); err != nil {
		return err
	}
	return nil
}

func validateFields(section string, fields []Field) error {
	seen := map[string]bool{}
	for _, f := range fields {
		if f.Name == "" {
			return fmt.Errorf("%s field name is required", section)
		}
		if seen[f.Name] {
			return fmt.Errorf("duplicate %s field %q", section, f.Name)
		}
		seen[f.Name] = true

		switch f.Type {
		case "string", "number", "boolean", "array", "object":
		default:
			return fmt.Errorf("%s field %q has unsupported type %q", section, f.Name, f.Type)
		}
		if f.Type != "array" && f.ItemsType != "" {
			return fmt.Errorf("%s field %q has items_type but is not array", section, f.Name)
		}
	}
	return nil
}

// ValidatePayload validates payload fields against contract fields.
// If strict is true, unknown fields are reported as errors.
func ValidatePayload(fields []Field, payload map[string]any, strict bool) []ValidationError {
	var errs []ValidationError
	index := map[string]Field{}
	for _, f := range fields {
		index[f.Name] = f
	}

	for _, f := range fields {
		val, found := payload[f.Name]
		if f.Required && !found {
			errs = append(errs, ValidationError{
				Field:   f.Name,
				Message: "required field is missing",
			})
			continue
		}
		if !found {
			continue
		}
		if !matchesType(val, f.Type) {
			errs = append(errs, ValidationError{
				Field:   f.Name,
				Message: fmt.Sprintf("expected %s, got %T", f.Type, val),
			})
			continue
		}
		if f.Type == "array" && f.ItemsType != "" {
			arr, _ := toSlice(val)
			for i, item := range arr {
				if !matchesType(item, f.ItemsType) {
					errs = append(errs, ValidationError{
						Field:   f.Name,
						Message: fmt.Sprintf("item %d expected %s, got %T", i, f.ItemsType, item),
					})
					continue
				}
				if len(f.Enum) > 0 && f.ItemsType == "string" {
					s, _ := item.(string)
					if !contains(f.Enum, s) {
						errs = append(errs, ValidationError{
							Field:   f.Name,
							Message: fmt.Sprintf("item %d value %q not in enum", i, s),
						})
					}
				}
			}
		} else if len(f.Enum) > 0 {
			s, ok := val.(string)
			if !ok {
				errs = append(errs, ValidationError{
					Field:   f.Name,
					Message: "enum is only supported for string fields or string arrays",
				})
				continue
			}
			if !contains(f.Enum, s) {
				errs = append(errs, ValidationError{
					Field:   f.Name,
					Message: fmt.Sprintf("value %q not in enum", s),
				})
			}
		}
	}

	if strict {
		for key := range payload {
			if _, ok := index[key]; !ok {
				errs = append(errs, ValidationError{
					Field:   key,
					Message: "unknown field",
				})
			}
		}
	}
	return errs
}

func contains(values []string, candidate string) bool {
	for _, v := range values {
		if v == candidate {
			return true
		}
	}
	return false
}

func matchesType(v any, t string) bool {
	switch t {
	case "string":
		_, ok := v.(string)
		return ok
	case "number":
		switch v.(type) {
		case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := toSlice(v)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		if ok {
			return true
		}
		rv := reflect.ValueOf(v)
		return rv.IsValid() && rv.Kind() == reflect.Struct
	default:
		return false
	}
}

func toSlice(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}
