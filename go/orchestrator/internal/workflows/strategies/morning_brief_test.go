package strategies

import (
	"encoding/json"
	"testing"
)

func TestExtractNewsCount(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		// Default when no count specified
		{"デジタルマーケティングの最新ニュース", 10},
		{"latest marketing news", 10},
		{"", 10},

		// Japanese: N件
		{"ニュースを15件表示して", 15},
		{"5件のニュースをまとめて", 5},
		{"トップ7件のニュース", 7},

		// Japanese: N個
		{"ニュースを3個ください", 3},

		// Japanese: トップN
		{"トップ12のニュースを教えて", 12},

		// English: top N
		{"top 8 news for today", 8},
		{"show me top 5 items", 5},

		// English: N news/items
		{"give me 15 news items", 15},
		{"show 7 news", 7},

		// Clamped to 20
		{"ニュースを50件", 20},
		{"top 100 news", 20},

		// Clamped to 1 minimum
		{"0件のニュース", 10}, // 0 → default
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := extractNewsCount(tt.query)
			if got != tt.want {
				t.Errorf("extractNewsCount(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	validObj := `{"items":[{"headline":"test"}],"summary":"ok"}`

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean JSON",
			input: validObj,
			want:  validObj,
		},
		{
			name:  "markdown fence json",
			input: "```json\n" + validObj + "\n```",
			want:  validObj,
		},
		{
			name:  "markdown fence no lang",
			input: "```\n" + validObj + "\n```",
			want:  validObj,
		},
		{
			name:  "preamble text before JSON",
			input: "Here is the result:\n" + validObj,
			want:  validObj,
		},
		{
			name:  "trailing text after JSON",
			input: validObj + "\n\nNote: all items verified.",
			want:  validObj,
		},
		{
			name:  "preamble with braces before real JSON",
			input: "Example: {x} is invalid. Actual output:\n" + validObj,
			want:  validObj,
		},
		{
			name:  "fence plus preamble",
			input: "```json\nHere it is:\n" + validObj + "\n```",
			want:  validObj,
		},
		{
			name:  "whitespace padded",
			input: "  \n\n  " + validObj + "  \n\n  ",
			want:  validObj,
		},
		{
			name:  "completely invalid — returns best effort",
			input: "no json here at all",
			want:  "no json here at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() =\n  %q\nwant\n  %q", got, tt.want)
			}
		})
	}
}

func TestResolvedItems(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantCount int
		wantFirst string // headline of first item
	}{
		{
			name:      "canonical items key",
			jsonInput: `{"items":[{"headline":"A"},{"headline":"B"}],"summary":"s"}`,
			wantCount: 2,
			wantFirst: "A",
		},
		{
			name:      "alternate news_items key",
			jsonInput: `{"news_items":[{"headline":"X"}],"summary":"s"}`,
			wantCount: 1,
			wantFirst: "X",
		},
		{
			name:      "both keys — items wins",
			jsonInput: `{"items":[{"headline":"A"}],"news_items":[{"headline":"X"}],"summary":"s"}`,
			wantCount: 1,
			wantFirst: "A",
		},
		{
			name:      "both keys — items empty, news_items used",
			jsonInput: `{"items":[],"news_items":[{"headline":"X"}],"summary":"s"}`,
			wantCount: 1,
			wantFirst: "X",
		},
		{
			name:      "neither key present",
			jsonInput: `{"summary":"nothing"}`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out MorningBriefOutput
			if err := json.Unmarshal([]byte(tt.jsonInput), &out); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			items := out.resolvedItems()
			if len(items) != tt.wantCount {
				t.Fatalf("resolvedItems() count = %d, want %d", len(items), tt.wantCount)
			}
			if tt.wantCount > 0 && items[0].Headline != tt.wantFirst {
				t.Errorf("first headline = %q, want %q", items[0].Headline, tt.wantFirst)
			}
		})
	}
}

func TestRecommendedActionsParsing(t *testing.T) {
	input := `{
		"items": [{"headline":"A","insight":"i","source":"s","source_url":"","category":"c"}],
		"summary": "sum",
		"recommended_actions": [
			{"priority":"high","action":"Update SEO strategy","rationale":"Competitor changed pricing"},
			{"priority":"low","action":"Monitor trend","rationale":"Minor shift observed"}
		]
	}`

	var out MorningBriefOutput
	if err := json.Unmarshal([]byte(input), &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(out.RecommendedActions) != 2 {
		t.Fatalf("RecommendedActions count = %d, want 2", len(out.RecommendedActions))
	}
	if out.RecommendedActions[0].Priority != "high" {
		t.Errorf("first action priority = %q, want %q", out.RecommendedActions[0].Priority, "high")
	}
	if out.RecommendedActions[1].Action != "Monitor trend" {
		t.Errorf("second action = %q", out.RecommendedActions[1].Action)
	}

	// Without recommended_actions — should still parse fine
	input2 := `{"items":[{"headline":"A"}],"summary":"s"}`
	var out2 MorningBriefOutput
	if err := json.Unmarshal([]byte(input2), &out2); err != nil {
		t.Fatalf("Unmarshal without actions error: %v", err)
	}
	if len(out2.RecommendedActions) != 0 {
		t.Errorf("RecommendedActions should be empty, got %d", len(out2.RecommendedActions))
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short string = %q", got)
	}
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("truncate long string = %q", got)
	}
}
