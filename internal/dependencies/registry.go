package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

const defaultRegistry = "https://registry.npmjs.org"

type registryClient struct {
	baseURL string
	http    *http.Client
	cache   map[string]packageDocument
}

type packageDocument struct {
	Name     string                    `json:"name"`
	Tags     map[string]string         `json:"dist-tags"`
	Versions map[string]packageVersion `json:"versions"`
}

type packageVersion struct {
	Name                 string                        `json:"name"`
	Version              string                        `json:"version"`
	Dependencies         map[string]string             `json:"dependencies"`
	PeerDependencies     map[string]string             `json:"peerDependencies"`
	PeerDependenciesMeta map[string]peerDependencyMeta `json:"peerDependenciesMeta"`
	HasInstallScript     bool                          `json:"hasInstallScript"`
	Dist                 packageDistribution           `json:"dist"`
}

type peerDependencyMeta struct {
	Optional bool `json:"optional"`
}

type packageDistribution struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
	Shasum    string `json:"shasum"`
}

func newRegistryClient(registry string, client *http.Client) *registryClient {
	if strings.TrimSpace(registry) == "" {
		registry = defaultRegistry
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &registryClient{baseURL: strings.TrimRight(registry, "/"), http: client, cache: map[string]packageDocument{}}
}

func (client *registryClient) resolve(ctx context.Context, name, constraint string) (packageVersion, error) {
	document, err := client.packageMetadata(ctx, name)
	if err != nil {
		return packageVersion{}, err
	}
	selected := constraint
	if tagged := document.Tags[constraint]; tagged != "" {
		selected = tagged
	}
	if exact, ok := document.Versions[selected]; ok {
		return exact, nil
	}
	rule, err := semver.NewConstraint(constraint)
	if err != nil {
		return packageVersion{}, fmt.Errorf("dependency %s has unsupported version %q", name, constraint)
	}
	versions := make([]*semver.Version, 0, len(document.Versions))
	for raw := range document.Versions {
		version, parseErr := semver.NewVersion(raw)
		if parseErr == nil && rule.Check(version) {
			versions = append(versions, version)
		}
	}
	if len(versions) == 0 {
		return packageVersion{}, fmt.Errorf("no version of %s satisfies %q", name, constraint)
	}
	sort.Sort(sort.Reverse(semver.Collection(versions)))
	return document.Versions[versions[0].Original()], nil
}

func (client *registryClient) packageMetadata(ctx context.Context, name string) (packageDocument, error) {
	if cached, ok := client.cache[name]; ok {
		return cached, nil
	}
	endpoint := client.baseURL + "/" + url.PathEscape(name)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return packageDocument{}, err
	}
	request.Header.Set("Accept", "application/vnd.npm.install-v1+json; q=1.0, application/json; q=0.8")
	request.Header.Set("User-Agent", "Northframe/0.9")
	response, err := client.http.Do(request)
	if err != nil {
		return packageDocument{}, fmt.Errorf("fetch %s metadata: %w", name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return packageDocument{}, fmt.Errorf("fetch %s metadata: registry returned %s", name, response.Status)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 32<<20))
	var document packageDocument
	if err := decoder.Decode(&document); err != nil {
		return packageDocument{}, fmt.Errorf("decode %s metadata: %w", name, err)
	}
	if document.Name == "" || len(document.Versions) == 0 {
		return packageDocument{}, fmt.Errorf("registry returned incomplete metadata for %s", name)
	}
	client.cache[name] = document
	return document, nil
}
