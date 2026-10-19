package st

// Only bclng001 extends option names to include hyphens. Other ExMode command
// parsers retain their existing grammar.
const bucklingNamedArgPattern = "^ *-{1,2}([a-zA-Z][a-zA-Z-]*)(={0,1})([^ =]*) *$"
