# Internal admin

`pkg/admin` mounts a framework-owned internal administration UI for registered resources. It provides listing, search, pagination, create, edit, delete, field validation, and permission checks while leaving persistence and business rules in application services.

Use it for trusted staff operations. It is not a storefront page builder and it does not generate a database schema.

## Implement a repository

The admin UI depends on a small CRUD boundary:

```go
type productAdminRepository struct {
    service *catalog.Service
}

func (repo productAdminRepository) List(
    ctx context.Context,
    query admin.ListQuery,
) (admin.Page, error) {
    result, err := repo.service.AdminList(ctx, catalog.AdminListInput{
        Search: query.Search,
        Sort: query.Sort,
        Desc: query.Desc,
        Page: query.Page,
        PageSize: query.PageSize,
        Filters: query.Filters,
    })
    if err != nil {
        return admin.Page{}, err
    }
    return admin.Page{
        Records: productRecords(result.Products),
        Total: result.Total,
    }, nil
}
```

Implement `Get`, `Create`, `Update`, and `Delete` through the same service. Repository methods receive the standard context, so sqlc, transactions, tenant scope, and auditing remain application concerns.

`admin.Record` is `map[string]any` because resources can describe different field sets. Convert it into typed service input at the repository boundary rather than passing maps into domain code.

## Register a resource

```go
registry := admin.NewRegistry()
registry.MustRegister(admin.Resource{
    Name:        "products",
    Label:       "Product",
    PluralLabel: "Products",
    Icon:        "solar:box-bold",
    Description: "Manage the product catalogue.",
    Repository:  productAdminRepository{service: catalogService},
    Fields: []admin.Field{
        {
            Name: "id", Label: "ID",
            ReadOnly: true,
        },
        {
            Name: "name", Label: "Product name",
            Required: true, Searchable: true, Sortable: true,
        },
        {
            Name: "price_cents", Label: "Price",
            Kind: admin.FieldMoney, Required: true, Sortable: true,
        },
        {
            Name: "status", Label: "Status",
            Kind: admin.FieldSelect,
            Options: []admin.Option{
                {Value: "draft", Label: "Draft"},
                {Value: "active", Label: "Active"},
                {Value: "archived", Label: "Archived"},
            },
        },
    },
    Validate: func(ctx context.Context, record admin.Record) web.FieldErrors {
        errors := web.FieldErrors{}
        if strings.TrimSpace(stringValue(record["name"])) == "" {
            errors["name"] = "Product name is required"
        }
        return errors
    },
})
```

Supported field kinds include text, long text, number, boolean, date, datetime, money, select, and relation. A field may be required, read-only, hidden, searchable, or sortable. The repository remains responsible for validating sort fields and scoping every operation to the current tenant.

## Mount with Northframe sessions

```go
if err := admin.Mount(app, registry, sessions, admin.Options{
    BasePath:  "/admin",
    LoginPath: "/login",
    Title:     "TopDuka Administration",
    PageSize:  50,
}); err != nil {
    log.Fatal(err)
}
```

The mount adds CSRF-protected routes under `/admin`. Anonymous users are rejected or redirected, and every resource operation checks its permission.

Default permission names are:

- `admin.products.view`
- `admin.products.create`
- `admin.products.update`
- `admin.products.delete`

Override them with `admin.Permissions` when the application already has a permission vocabulary.

## Use an existing authentication system

`admin.MountWithAccess` accepts application middleware and a resolver:

```go
err := admin.MountWithAccess(app, registry, admin.Access{
    Middleware: []web.Middleware{requireStaff},
    Session: func(ctx *web.Context) (auth.Session, bool) {
        account, ok := currentAccount(ctx)
        if !ok {
            return auth.Session{}, false
        }
        return auth.Session{
            Subject: account.ID,
            Permissions: account.Permissions,
        }, true
    },
}, admin.Options{BasePath: "/internal"})
```

This maps existing cookies, JWTs, or staff middleware into the permission session expected by admin. It does not require migrating the application's authentication tables.

## Production responsibilities

Admin routes are powerful. Require strong staff authentication, least-privilege permissions, CSRF, secure cookies, audit events, tenant scoping, and rate limits where appropriate. A permission check in navigation is not enough; the mounted handlers enforce operation permissions, and repositories must still enforce record-level access.

Test each resource with allowed and denied sessions, invalid select values, validation failures, search, pagination, cross-tenant identifiers, and delete behavior. Keep destructive operations explicit and recoverable when the product domain permits it.
