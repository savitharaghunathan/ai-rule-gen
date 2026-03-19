package generation

import "fmt"

// IDGenerator produces rule IDs in the format <prefix>-<number>,
// incrementing by 10 to leave room for manual insertions.
type IDGenerator struct {
	prefix  string
	current int
}

// NewIDGenerator creates a generator with the given prefix.
// IDs start at 00010 and increment by 10.
func NewIDGenerator(prefix string) *IDGenerator {
	return &IDGenerator{prefix: prefix, current: 0}
}

// Next returns the next rule ID.
func (g *IDGenerator) Next() string {
	g.current += 10
	return fmt.Sprintf("%s-%05d", g.prefix, g.current)
}
