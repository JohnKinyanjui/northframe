package help

import "testing"

func TestQuestionBankIsComprehensive(t *testing.T) {
	if len(questionBank) < 25 {
		t.Fatalf("question bank has %d entries, want at least 25", len(questionBank))
	}
}

func TestFilterQuestionsSearchesAllContent(t *testing.T) {
	for _, test := range []struct {
		query string
		want  int
	}{
		{query: "svelte", want: 1},
		{query: "webassembly", want: 1},
		{query: "package.json", want: 1},
		{query: "components", want: 1},
		{query: "does-not-exist", want: 0},
	} {
		if got := len(filterQuestions(test.query)); (test.want == 0 && got != 0) || (test.want > 0 && got < test.want) {
			t.Errorf("filterQuestions(%q) returned %d entries", test.query, got)
		}
	}
}
