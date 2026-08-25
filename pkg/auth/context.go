package auth

import "context"

type contextSessionKey struct{}

// WithSession stores authenticated state in a standard Go context for service
// and repository boundaries that do not depend on Northframe's web context.
func WithSession(ctx context.Context, session Session) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextSessionKey{}, cloneSession(session))
}

// SessionFromContext returns authentication state attached with WithSession.
func SessionFromContext(ctx context.Context) (Session, bool) {
	if ctx == nil {
		return Session{}, false
	}
	session, ok := ctx.Value(contextSessionKey{}).(Session)
	return cloneSession(session), ok
}
