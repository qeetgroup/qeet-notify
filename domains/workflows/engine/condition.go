package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// EvalCondition evaluates a tiny, safe boolean expression against the event
// payload. It deliberately supports only a minimal grammar (no eval, no deps):
//
//	<path> <op> <literal>   where op ∈ == != > < >= <=
//	<path>                  truthy test (non-nil, non-empty, non-zero, true)
//
// <path> is a dot-separated lookup into nested maps, e.g. "user.tier".
// <literal> is a quoted string ("gold" / 'gold'), a number (42, 3.14), or a
// bool (true/false). Numeric comparison is used when both sides are numeric
// (numeric strings included); otherwise string comparison is used.
func EvalCondition(expr string, payload map[string]any) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, fmt.Errorf("empty condition")
	}

	op, path, lit := splitExpr(expr)
	val := lookupPath(payload, path)

	if op == "" {
		return truthy(val), nil
	}
	return compare(val, op, lit)
}

// splitExpr finds the comparison operator and returns (op, lhs path, rhs literal).
// Two-character operators are matched before single-character ones so ">=" is
// never mistaken for ">".
func splitExpr(expr string) (op, path, lit string) {
	for _, o := range []string{"==", "!=", ">=", "<="} {
		if idx := strings.Index(expr, o); idx >= 0 {
			return o, strings.TrimSpace(expr[:idx]), strings.TrimSpace(expr[idx+2:])
		}
	}
	for _, o := range []string{">", "<"} {
		if idx := strings.Index(expr, o); idx >= 0 {
			return o, strings.TrimSpace(expr[:idx]), strings.TrimSpace(expr[idx+1:])
		}
	}
	return "", strings.TrimSpace(expr), ""
}

// lookupPath walks a dot-separated path into nested map[string]any values.
func lookupPath(payload map[string]any, path string) any {
	if path == "" {
		return nil
	}
	var cur any = payload
	for _, p := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

func compare(val any, op, lit string) (bool, error) {
	// Prefer numeric comparison when both operands are numeric.
	if litNum, ok := parseNumber(lit); ok {
		if valNum, ok := toNumber(val); ok {
			switch op {
			case "==":
				return valNum == litNum, nil
			case "!=":
				return valNum != litNum, nil
			case ">":
				return valNum > litNum, nil
			case "<":
				return valNum < litNum, nil
			case ">=":
				return valNum >= litNum, nil
			case "<=":
				return valNum <= litNum, nil
			}
		}
	}

	// Boolean comparison.
	if lit == "true" || lit == "false" {
		if vb, ok := val.(bool); ok {
			switch op {
			case "==":
				return vb == (lit == "true"), nil
			case "!=":
				return vb != (lit == "true"), nil
			}
		}
	}

	// Fall back to string comparison.
	litStr := unquote(lit)
	valStr := toString(val)
	switch op {
	case "==":
		return valStr == litStr, nil
	case "!=":
		return valStr != litStr, nil
	case ">":
		return valStr > litStr, nil
	case "<":
		return valStr < litStr, nil
	case ">=":
		return valStr >= litStr, nil
	case "<=":
		return valStr <= litStr, nil
	}
	return false, fmt.Errorf("unsupported operator %q", op)
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return true // non-nil maps, slices, etc.
	}
}

// parseNumber parses a bare numeric literal (no surrounding quotes).
func parseNumber(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// toNumber coerces a payload value to a float64 where possible (JSON numbers
// decode as float64; numeric strings are accepted too).
func toNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// unquote strips a single matching pair of surrounding quotes, if present.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
