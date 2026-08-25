package dependencies

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerAddsLocksMaterializesAndRemovesPackages(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/app\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tarballs := map[string][]byte{
		"tiny":   packageTarball(t, "tiny", "1.0.0", `export const answer = 42;`),
		"helper": packageTarball(t, "helper", "1.2.0", `export const help = true;`),
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/tiny", "/helper":
			name := request.URL.Path[1:]
			version := "1.0.0"
			dependencies := map[string]string{}
			if name == "tiny" {
				dependencies["helper"] = "^1.0.0"
			} else {
				version = "1.2.0"
			}
			writePackageDocument(t, writer, serverURL(request), name, version, dependencies, tarballs[name])
		case "/tar/tiny.tgz":
			_, _ = writer.Write(tarballs["tiny"])
		case "/tar/helper.tgz":
			_, _ = writer.Write(tarballs["helper"])
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	manager := Manager{Root: root, Registry: server.URL, Client: server.Client()}
	lock, err := manager.Add(context.Background(), []string{"tiny"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Packages) != 2 || lock.Packages["tiny"].Version != "1.0.0" || lock.Packages["helper"].Version != "1.2.0" {
		t.Fatalf("lock = %#v", lock)
	}
	for _, name := range []string{"tiny", "helper"} {
		if _, err := os.Stat(filepath.Join(root, ".northframe", "modules", "node_modules", name, "package.json")); err != nil {
			t.Fatalf("%s was not materialized: %v", name, err)
		}
	}
	manifest, err := LoadManifest(root)
	if err != nil || manifest.Client.Dependencies["tiny"] != "latest" {
		t.Fatalf("manifest = %#v, err = %v", manifest, err)
	}
	project, err := InspectProject(root)
	if err != nil || project.Dependencies["tiny"] != "latest" {
		t.Fatalf("project = %#v, err = %v", project, err)
	}

	lock, err = manager.Remove(context.Background(), []string{"tiny"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Packages) != 0 {
		t.Fatalf("removed lock = %#v", lock)
	}
}

func TestVerifyIntegrityRejectsChangedArchive(t *testing.T) {
	contents := []byte("original")
	digest := sha512.Sum512(contents)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(digest[:])
	if err := verifyIntegrity([]byte("changed"), integrity); err == nil {
		t.Fatal("changed archive passed integrity verification")
	}
}

func TestParsePackageSpecSupportsScopedPackages(t *testing.T) {
	spec, err := ParsePackageSpec("@scope/tool@^2.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "@scope/tool" || spec.Constraint != "^2.1.0" {
		t.Fatalf("spec = %#v", spec)
	}
	if PackageName("@scope/tool/browser") != "@scope/tool" {
		t.Fatal("scoped import subpath was not normalized")
	}
}

func writePackageDocument(t *testing.T, writer http.ResponseWriter, baseURL, name, version string, dependencies map[string]string, tarball []byte) {
	t.Helper()
	digest := sha512.Sum512(tarball)
	document := packageDocument{
		Name: name, Tags: map[string]string{"latest": version},
		Versions: map[string]packageVersion{version: {
			Name: name, Version: version, Dependencies: dependencies,
			Dist: packageDistribution{
				Tarball:   baseURL + "/tar/" + name + ".tgz",
				Integrity: "sha512-" + base64.StdEncoding.EncodeToString(digest[:]),
			},
		}},
	}
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(document); err != nil {
		t.Fatal(err)
	}
}

func packageTarball(t *testing.T, name, version, source string) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	files := map[string]string{
		"package/package.json": fmt.Sprintf(`{"name":%q,"version":%q,"module":"index.js"}`, name, version),
		"package/index.js":     source,
	}
	for name, contents := range files {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(contents))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func serverURL(request *http.Request) string {
	return "http://" + request.Host
}
