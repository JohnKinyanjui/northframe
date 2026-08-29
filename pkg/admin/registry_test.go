package admin

import (
	"context"
	"testing"

	"github.com/JohnKinyanjui/northframe/pkg/auth"
)

type testRepository struct{}

func (testRepository) List(context.Context, ListQuery) (Page, error)  { return Page{}, nil }
func (testRepository) Get(context.Context, string) (Record, error)    { return Record{}, nil }
func (testRepository) Create(context.Context, Record) (Record, error) { return Record{}, nil }
func (testRepository) Update(context.Context, string, Record) (Record, error) {
	return Record{}, nil
}
func (testRepository) Delete(context.Context, string) error { return nil }

func TestRegistryNormalizesAndFiltersResources(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Resource{
		Name: "staff-members", Repository: testRepository{},
		Fields: []Field{{Name: "email", Searchable: true}, {Name: "role", Kind: FieldSelect}},
	}); err != nil {
		t.Fatal(err)
	}
	resource, ok := registry.Resource("staff-members")
	if !ok {
		t.Fatal("registered resource is missing")
	}
	if resource.Label != "Staff Members" || resource.Permissions.View != "admin.staff.members.view" {
		t.Fatalf("unexpected resource: %#v", resource)
	}
	visible := registry.VisibleResources(auth.Session{Permissions: []string{"admin.staff.*"}})
	if len(visible) != 1 || visible[0].Name != "staff-members" {
		t.Fatalf("visible resources = %#v", visible)
	}
}

func TestRegistryRejectsInvalidAndDuplicateResources(t *testing.T) {
	registry := NewRegistry()
	resource := Resource{Name: "orders", Repository: testRepository{}, Fields: []Field{{Name: "id"}}}
	if err := registry.Register(resource); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(resource); err == nil {
		t.Fatal("duplicate resource was accepted")
	}
	if err := registry.Register(Resource{Name: "Bad Name", Repository: testRepository{}, Fields: []Field{{Name: "id"}}}); err == nil {
		t.Fatal("invalid resource name was accepted")
	}
}
