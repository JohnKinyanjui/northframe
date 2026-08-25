package web

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteClientPropsProducesSafeInertJSON(t *testing.T) {
	var output bytes.Buffer
	if err := WriteClientProps(&output, "inventory-page", map[string]string{"name": `</script><img src=x>`}); err != nil {
		t.Fatal(err)
	}
	result := output.String()
	if !strings.Contains(result, `data-north-props="inventory-page"`) {
		t.Fatalf("missing props marker: %s", result)
	}
	if strings.Contains(result, `</script><img`) || !strings.Contains(result, `\u003c/script\u003e`) {
		t.Fatalf("client props were not HTML-safe: %s", result)
	}
}
