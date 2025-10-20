package runtime

import (
	"fmt"
	"strings"
)

// EvalCondition evaluates a simple condition like "{{var}} == 'hello'".
func EvalCondition(ctx interface{ Get(string) string }, condition string) bool {
	// Remove whitespace
	cond := strings.TrimSpace(condition)

	// Support == comparison
	if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		// Extract variable in {{ }}
		if strings.HasPrefix(left, "{{") && strings.HasSuffix(left, "}}") {
			varName := strings.TrimSpace(left[2 : len(left)-2])
			leftVal := ctx.Get(varName)

			// Remove quotes from right side
			rightVal := strings.Trim(right, `"'`)

			return leftVal == rightVal
		}
	}

	fmt.Printf("⚠️ Unsupported condition: %s\n", condition)
	return false
}
