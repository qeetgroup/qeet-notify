package engine

import "testing"

func TestEvalCondition(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		payload map[string]any
		want    bool
		wantErr bool
	}{
		{"string eq true", `tier == "gold"`, map[string]any{"tier": "gold"}, true, false},
		{"string eq false", `tier == "gold"`, map[string]any{"tier": "silver"}, false, false},
		{"string neq", `tier != "gold"`, map[string]any{"tier": "silver"}, true, false},
		{"single quotes", `tier == 'gold'`, map[string]any{"tier": "gold"}, true, false},
		{"number gt true", `age > 18`, map[string]any{"age": float64(21)}, true, false},
		{"number gt false", `age > 18`, map[string]any{"age": float64(15)}, false, false},
		{"number gte eq", `age >= 18`, map[string]any{"age": float64(18)}, true, false},
		{"number lte eq", `age <= 18`, map[string]any{"age": float64(18)}, true, false},
		{"number lt", `age < 18`, map[string]any{"age": float64(10)}, true, false},
		{"number eq zero", `count == 0`, map[string]any{"count": float64(0)}, true, false},
		{"numeric string coerced", `age > 18`, map[string]any{"age": "21"}, true, false},
		{"bool eq true", `active == true`, map[string]any{"active": true}, true, false},
		{"bool eq false", `active == false`, map[string]any{"active": false}, true, false},
		{"bool neq", `active != true`, map[string]any{"active": false}, true, false},
		{"nested path", `user.tier == "gold"`, map[string]any{"user": map[string]any{"tier": "gold"}}, true, false},
		{"nested missing", `user.tier == "gold"`, map[string]any{"user": map[string]any{}}, false, false},
		{"truthy string", `verified`, map[string]any{"verified": "yes"}, true, false},
		{"truthy empty string", `verified`, map[string]any{"verified": ""}, false, false},
		{"truthy bool", `verified`, map[string]any{"verified": true}, true, false},
		{"truthy missing", `verified`, map[string]any{}, false, false},
		{"truthy number nonzero", `n`, map[string]any{"n": float64(3)}, true, false},
		{"truthy number zero", `n`, map[string]any{"n": float64(0)}, false, false},
		{"empty expr errors", ``, map[string]any{}, false, true},
		{"whitespace expr errors", `   `, map[string]any{}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvalCondition(tt.expr, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EvalCondition(%q) err=%v, wantErr=%v", tt.expr, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("EvalCondition(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestNextIndex(t *testing.T) {
	idx := map[string]int{"a": 0, "b": 1, "c": 2}

	if got := nextIndex(idx, Step{NextStep: "c"}, 0); got != 2 {
		t.Errorf("NextStep known: want 2, got %d", got)
	}
	if got := nextIndex(idx, Step{NextStep: "missing"}, 0); got != 1 {
		t.Errorf("NextStep unknown: want fall-through 1, got %d", got)
	}
	if got := nextIndex(idx, Step{}, 1); got != 2 {
		t.Errorf("no NextStep: want i+1=2, got %d", got)
	}
}
