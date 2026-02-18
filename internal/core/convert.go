package core

// IntFromAny converts a numeric value (float64, int, or int64) to int, returning 0 for unsupported types.
func IntFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
