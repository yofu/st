package st

// bclng001 and vibeig extend option names to include hyphens for eigenanalysis
// diagnostics. Other ExMode command parsers retain their existing grammar.
const bucklingNamedArgPattern = "^ *-{1,2}([a-zA-Z][a-zA-Z-]*)(={0,1})([^ =]*) *$"
