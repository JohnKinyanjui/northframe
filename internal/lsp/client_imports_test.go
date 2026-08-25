package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientImportCompletionIncludesDependenciesAndLocalModules(t *testing.T) {
	root := t.TempDir()
	writeClientImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.27\n")
	writeClientImportTestFile(t, filepath.Join(root, "northframe.toml"), "[client]\nsource = 'browser'\n\n[client.dependencies]\n'date-fns' = 'latest'\n")
	writeClientImportTestFile(t, filepath.Join(root, "browser", "money.ts"), "export const money = true\n")
	component := filepath.Join(root, "routes", "page.north")
	text := `<script lang="ts">
import { format } from "d
</script>`
	writeClientImportTestFile(t, component, text)
	offset := len(`<script lang="ts">
import { format } from "d`)
	items := clientImportCompletionItems(documentURI(component), text, offset)
	if len(items) != 1 || items[0]["label"] != "date-fns" {
		t.Fatalf("dependency completion items = %#v", items)
	}

	text = `<script lang="ts">
import money from "$client/
</script>`
	offset = len(`<script lang="ts">
import money from "$client/`)
	items = clientImportCompletionItems(documentURI(component), text, offset)
	if len(items) != 1 || items[0]["label"] != "$client/money" {
		t.Fatalf("local completion items = %#v", items)
	}
}

func writeClientImportTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
