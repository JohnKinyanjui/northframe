# Authentication

Northframe provides opaque server-side sessions and permission middleware, but it does not prescribe a user table, password hashing library, OAuth provider, or account recovery flow. Your application verifies identity; `pkg/auth` owns the session cookie and request authentication state.

## Configure the session manager

```go
store := postgresSessionStore{DB: db}
sessions, err := auth.New(auth.Config{
    Store:      store,
    CookieName: "shop_session",
    CookiePath: "/",
    Lifetime:   24 * time.Hour,
    Secure:     production,
    SameSite:   http.SameSiteLaxMode,
})
if err != nil {
    log.Fatal(err)
}

app := web.New()
app.Use(auth.Load(sessions))
web.Provide(app, sessions)
```

Implement `auth.Store` with PostgreSQL, MySQL, SQLite, or a durable cache:

```go
type Store interface {
    Load(context.Context, string) (auth.Session, error)
    Save(context.Context, auth.Session) error
    Delete(context.Context, string) error
}
```

`auth.NewMemoryStore()` is useful for tests and local development, but sessions disappear when the process restarts and are not shared across replicas.

## Sign in

After verifying the submitted credentials, start the session:

```go
func login(ctx *web.Context, input LoginInput) (web.ActionResult, error) {
    account, err := accounts.VerifyPassword(
        ctx.StdContext(),
        strings.ToLower(strings.TrimSpace(input.Email)),
        input.Password,
    )
    if errors.Is(err, accounts.ErrInvalidCredentials) {
        return web.ActionInvalid("Invalid credentials", web.FieldErrors{
            "email": "Email or password is incorrect",
        }), nil
    }
    if err != nil {
        return web.ActionResult{}, err
    }

    sessions := web.MustUse[*auth.Manager](ctx)
    _, err = sessions.Start(ctx.StdContext(), ctx.Response, auth.SessionInput{
        Subject: account.ID,
        Values: map[string]string{
            "store_id": account.StoreID,
            "name":     account.Name,
        },
        Permissions: account.Permissions,
    })
    if err != nil {
        return web.ActionResult{}, err
    }
    return web.ActionRedirect("/dashboard", http.StatusSeeOther), nil
}
```

The browser receives a random 32-byte HttpOnly token. The store receives only its SHA-256 digest, subject, expiry, values, and permissions. Do not store the user's password, access token, or large profile document in session values.

Always return the same public message for an unknown email and a wrong password. This avoids turning the login form into an account-enumeration endpoint.

## Read the current account

With `auth.Load` or `auth.Require` middleware active:

```go
func Layout(ctx *web.Context) (generated.LayoutProps, error) {
    session, authenticated := auth.Current(ctx)
    if !authenticated {
        return generated.LayoutProps{SignedIn: false}, nil
    }
    return generated.LayoutProps{
        SignedIn: true,
        AccountID: session.Subject,
        Name:      session.Values["name"],
    }, nil
}
```

Use `auth.MustCurrent(ctx)` only inside a route that is guaranteed to be protected. It panics with an actionable configuration message if no session middleware loaded a session.

## Protect a route subtree

Put the guard on a dashboard layout:

```go
func LayoutMiddleware() []web.Middleware {
    sessions := applicationSessions()
    return []web.Middleware{
        auth.Require(sessions, auth.GuardOptions{
            LoginPath: "/login",
        }),
    }
}
```

Anonymous browser requests are redirected when `LoginPath` is configured. Without a login path they receive 401. A session missing a requested permission receives 403.

## Permissions

Protect a sensitive page or API:

```go
auth.Require(sessions, auth.GuardOptions{
    Permissions: []string{"orders.read", "orders.refund"},
})
```

Every listed permission is required. Exact values such as `orders.read` grant one operation. `orders.*` grants an entire namespace, and `*` grants everything. Normalize permissions at the account or role boundary and keep authorization checks close to the protected operation.

Navigation visibility is not authorization. Hiding a link in a layout improves the interface but does not protect its URL.

## Sign out

```go
func logout(ctx *web.Context) error {
    sessions := web.MustUse[*auth.Manager](ctx)
    if err := sessions.End(ctx.StdContext(), ctx.Response, ctx.Request); err != nil {
        return err
    }
    return ctx.Redirect("/login", http.StatusSeeOther)
}
```

Ending a session removes the stored digest and expires the cookie. For a “sign out everywhere” feature, application storage must find and revoke all sessions for the subject.

## Production checklist

Set `Secure: true` behind HTTPS. Use an HttpOnly cookie, a restrictive SameSite policy appropriate to the login flow, short enough expiry, CSRF protection on unsafe browser requests, rate limits on credential endpoints, and a slow password hash such as Argon2id or bcrypt. Rotate or revoke sessions after password changes and privilege changes. Test anonymous, expired, revoked, and under-permissioned sessions—not just the successful login.
