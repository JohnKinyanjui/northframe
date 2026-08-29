// Package admin defines the resource registry used by Northframe's internal
// administration UI. Applications keep persistence and business rules in
// their own services and adapt them through Repository.
package admin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/JohnKinyanjui/northframe/pkg/auth"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

type FieldKind string

const (
	FieldText     FieldKind = "text"
	FieldLongText FieldKind = "long_text"
	FieldNumber   FieldKind = "number"
	FieldBoolean  FieldKind = "boolean"
	FieldDate     FieldKind = "date"
	FieldDateTime FieldKind = "datetime"
	FieldMoney    FieldKind = "money"
	FieldSelect   FieldKind = "select"
	FieldRelation FieldKind = "relation"
)

type Field struct {
	Name       string
	Label      string
	Kind       FieldKind
	Required   bool
	ReadOnly   bool
	Hidden     bool
	Searchable bool
	Sortable   bool
	Options    []Option
}

type Option struct {
	Value string
	Label string
}

type Record map[string]any

type ListQuery struct {
	Search   string
	Sort     string
	Desc     bool
	Page     int
	PageSize int
	Filters  map[string]string
}

type Page struct {
	Records []Record
	Total   int64
}

// Repository is the service boundary consumed by the admin UI. Implementations
// should call application services or sqlc query sets instead of embedding SQL.
type Repository interface {
	List(context.Context, ListQuery) (Page, error)
	Get(context.Context, string) (Record, error)
	Create(context.Context, Record) (Record, error)
	Update(context.Context, string, Record) (Record, error)
	Delete(context.Context, string) error
}

type Permissions struct {
	View   string
	Create string
	Update string
	Delete string
}

type Resource struct {
	Name        string
	Label       string
	PluralLabel string
	Icon        string
	Description string
	Fields      []Field
	Permissions Permissions
	Repository  Repository
	Validate    func(context.Context, Record) web.FieldErrors
}

type Registry struct {
	mu        sync.RWMutex
	resources map[string]Resource
}

func NewRegistry() *Registry {
	return &Registry{resources: make(map[string]Resource)}
}

func (registry *Registry) Register(resource Resource) error {
	resource = normalizeResource(resource)
	if err := validateResource(resource); err != nil {
		return err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.resources[resource.Name]; exists {
		return fmt.Errorf("admin resource %q is already registered", resource.Name)
	}
	registry.resources[resource.Name] = cloneResource(resource)
	return nil
}

func (registry *Registry) MustRegister(resource Resource) {
	if err := registry.Register(resource); err != nil {
		panic("northframe: " + err.Error())
	}
}

func (registry *Registry) Resource(name string) (Resource, bool) {
	registry.mu.RLock()
	resource, ok := registry.resources[strings.ToLower(strings.TrimSpace(name))]
	registry.mu.RUnlock()
	return cloneResource(resource), ok
}

func (registry *Registry) Resources() []Resource {
	registry.mu.RLock()
	result := make([]Resource, 0, len(registry.resources))
	for _, resource := range registry.resources {
		result = append(result, cloneResource(resource))
	}
	registry.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].PluralLabel < result[j].PluralLabel })
	return result
}

// VisibleResources filters the registry through the authenticated session's
// view permissions.
func (registry *Registry) VisibleResources(session auth.Session) []Resource {
	resources := registry.Resources()
	visible := resources[:0]
	for _, resource := range resources {
		if session.Can(resource.Permissions.View) {
			visible = append(visible, resource)
		}
	}
	return visible
}

func normalizeResource(resource Resource) Resource {
	resource.Name = strings.ToLower(strings.TrimSpace(resource.Name))
	if resource.Label == "" {
		resource.Label = title(resource.Name)
	}
	if resource.PluralLabel == "" {
		resource.PluralLabel = resource.Label + "s"
	}
	prefix := "admin." + strings.ReplaceAll(resource.Name, "-", ".")
	if resource.Permissions.View == "" {
		resource.Permissions.View = prefix + ".view"
	}
	if resource.Permissions.Create == "" {
		resource.Permissions.Create = prefix + ".create"
	}
	if resource.Permissions.Update == "" {
		resource.Permissions.Update = prefix + ".update"
	}
	if resource.Permissions.Delete == "" {
		resource.Permissions.Delete = prefix + ".delete"
	}
	for index := range resource.Fields {
		resource.Fields[index].Name = strings.TrimSpace(resource.Fields[index].Name)
		if resource.Fields[index].Label == "" {
			resource.Fields[index].Label = title(resource.Fields[index].Name)
		}
		if resource.Fields[index].Kind == "" {
			resource.Fields[index].Kind = FieldText
		}
	}
	return resource
}

func validateResource(resource Resource) error {
	if resource.Name == "" {
		return errors.New("admin resource name is required")
	}
	for _, current := range resource.Name {
		if current < 'a' || current > 'z' {
			if current < '0' || current > '9' {
				if current != '-' {
					return fmt.Errorf("admin resource name %q must use lowercase letters, numbers, and hyphens", resource.Name)
				}
			}
		}
	}
	if resource.Repository == nil {
		return fmt.Errorf("admin resource %q repository is required", resource.Name)
	}
	if len(resource.Fields) == 0 {
		return fmt.Errorf("admin resource %q must declare at least one field", resource.Name)
	}
	seen := make(map[string]struct{}, len(resource.Fields))
	for _, field := range resource.Fields {
		if field.Name == "" {
			return fmt.Errorf("admin resource %q has a field without a name", resource.Name)
		}
		if _, exists := seen[field.Name]; exists {
			return fmt.Errorf("admin resource %q declares field %q more than once", resource.Name, field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return nil
}

func cloneResource(resource Resource) Resource {
	resource.Fields = append([]Field(nil), resource.Fields...)
	for index := range resource.Fields {
		resource.Fields[index].Options = append([]Option(nil), resource.Fields[index].Options...)
	}
	return resource
}

func title(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	words := strings.Fields(value)
	for index, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}
