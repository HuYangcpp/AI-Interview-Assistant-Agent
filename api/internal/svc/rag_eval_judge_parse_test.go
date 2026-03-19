package svc

import "testing"

func TestParseJudgeScore(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int
	}{
		{name: "plain digit", raw: "3", want: 3},
		{name: "score label", raw: "Score: 4", want: 4},
		{name: "json", raw: "{\"score\":5}", want: 5},
		{name: "json with spaces", raw: "{ \"score\" : 2 }", want: 2},
		{name: "chinese label", raw: "分数：1", want: 1},
		{name: "full width digit", raw: "评分：５", want: 5},
		{name: "chinese numeral", raw: "五", want: 5},
		{name: "markdown fenced json", raw: "```json\n{\"score\":4}\n```", want: 4},
		{name: "fraction style", raw: "4/5", want: 4},
		{name: "invalid zero", raw: "0", want: 0},
		{name: "invalid empty", raw: "", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseJudgeScore(tc.raw)
			if got != tc.want {
				t.Fatalf("parseJudgeScore(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}
