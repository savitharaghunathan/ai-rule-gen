package rules

// NewJavaReferenced creates a java.referenced condition.
func NewJavaReferenced(pattern, location string) Condition {
	return Condition{
		JavaReferenced: &JavaReferenced{
			Pattern:  pattern,
			Location: location,
		},
	}
}

// NewJavaReferencedAnnotated creates a java.referenced condition with an annotation filter.
func NewJavaReferencedAnnotated(pattern, location string, annotated *Annotated) Condition {
	return Condition{
		JavaReferenced: &JavaReferenced{
			Pattern:   pattern,
			Location:  location,
			Annotated: annotated,
		},
	}
}

// NewJavaDependency creates a java.dependency condition.
func NewJavaDependency(name, lowerbound, upperbound string) Condition {
	return Condition{
		JavaDependency: &Dependency{
			Name:       name,
			Lowerbound: lowerbound,
			Upperbound: upperbound,
		},
	}
}

// NewGoReferenced creates a go.referenced condition.
func NewGoReferenced(pattern string) Condition {
	return Condition{
		GoReferenced: &GoReferenced{
			Pattern: pattern,
		},
	}
}

// NewGoDependency creates a go.dependency condition.
func NewGoDependency(name, lowerbound, upperbound string) Condition {
	return Condition{
		GoDependency: &Dependency{
			Name:       name,
			Lowerbound: lowerbound,
			Upperbound: upperbound,
		},
	}
}

// NewNodejsReferenced creates a nodejs.referenced condition.
func NewNodejsReferenced(pattern string) Condition {
	return Condition{
		NodejsReferenced: &NodejsReferenced{
			Pattern: pattern,
		},
	}
}

// NewCSharpReferenced creates a csharp.referenced condition.
func NewCSharpReferenced(pattern, location string) Condition {
	return Condition{
		CSharpReferenced: &CSharpReferenced{
			Pattern:  pattern,
			Location: location,
		},
	}
}

// NewBuiltinFilecontent creates a builtin.filecontent condition.
func NewBuiltinFilecontent(pattern, filePattern string) Condition {
	return Condition{
		BuiltinFilecontent: &BuiltinFilecontent{
			Pattern:     pattern,
			FilePattern: filePattern,
		},
	}
}

// NewBuiltinFile creates a builtin.file condition.
func NewBuiltinFile(pattern string) Condition {
	return Condition{
		BuiltinFile: &BuiltinFile{
			Pattern: pattern,
		},
	}
}

// NewBuiltinXML creates a builtin.xml condition.
func NewBuiltinXML(xpath string, namespaces map[string]string) Condition {
	return Condition{
		BuiltinXML: &BuiltinXML{
			XPath:      xpath,
			Namespaces: namespaces,
		},
	}
}

// NewBuiltinJSON creates a builtin.json condition.
func NewBuiltinJSON(xpath string) Condition {
	return Condition{
		BuiltinJSON: &BuiltinJSON{
			XPath: xpath,
		},
	}
}

// NewBuiltinHasTags creates a builtin.hasTags condition.
func NewBuiltinHasTags(tags []string) Condition {
	return Condition{
		BuiltinHasTags: tags,
	}
}

// NewBuiltinXMLPublicID creates a builtin.xmlPublicID condition.
func NewBuiltinXMLPublicID(regex string, namespaces map[string]string) Condition {
	return Condition{
		BuiltinXMLPublicID: &BuiltinXMLPublicID{
			Regex:      regex,
			Namespaces: namespaces,
		},
	}
}

// NewOr creates an or combinator condition.
func NewOr(conditions ...Condition) Condition {
	entries := make([]ConditionEntry, len(conditions))
	for i, c := range conditions {
		entries[i] = ConditionEntry{Condition: c}
	}
	return Condition{Or: entries}
}

// NewAnd creates an and combinator condition.
func NewAnd(conditions ...Condition) Condition {
	entries := make([]ConditionEntry, len(conditions))
	for i, c := range conditions {
		entries[i] = ConditionEntry{Condition: c}
	}
	return Condition{And: entries}
}

// WithFrom adds a "from" chaining field to a condition.
func (c Condition) WithFrom(from string) Condition {
	c.From = from
	return c
}

// WithAs adds an "as" chaining field to a condition.
func (c Condition) WithAs(as string) Condition {
	c.As = as
	return c
}

// WithIgnore sets the "ignore" chaining field on a condition.
func (c Condition) WithIgnore(ignore bool) Condition {
	c.Ignore = &ignore
	return c
}

// WithNot sets the "not" chaining field on a condition.
func (c Condition) WithNot(not bool) Condition {
	c.Not = &not
	return c
}
