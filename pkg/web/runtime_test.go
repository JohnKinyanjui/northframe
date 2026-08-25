package web

import (
	"bytes"
	"testing"
)

func TestWriteEscaped(t *testing.T) {
	var output bytes.Buffer
	if err := WriteEscaped(&output, `<script>alert("no")</script>`); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), `&lt;script&gt;alert(&#34;no&#34;)&lt;/script&gt;`; got != want {
		t.Fatalf("WriteEscaped() = %q, want %q", got, want)
	}
}

func TestTruthy(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		want  bool
	}{
		{name: "nil", value: nil, want: false},
		{name: "empty string", value: "", want: false},
		{name: "non-empty slice", value: []string{"item"}, want: true},
		{name: "zero", value: 0, want: false},
		{name: "true", value: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Truthy(test.value); got != test.want {
				t.Fatalf("Truthy(%v) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
