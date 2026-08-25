package auth

import (
	"sort"
	"strings"
)

// Can reports whether a session grants a permission. A literal "*" grants
// everything; a suffix wildcard such as "orders.*" grants that namespace.
func (session Session) Can(permission string) bool {
	permission = normalizePermission(permission)
	if permission == "" {
		return false
	}
	for _, granted := range session.Permissions {
		if granted == "*" || granted == permission {
			return true
		}
		if strings.HasSuffix(granted, ".*") && strings.HasPrefix(permission, strings.TrimSuffix(granted, "*")) {
			return true
		}
	}
	return false
}

func (session Session) CanAll(permissions ...string) bool {
	for _, permission := range permissions {
		if !session.Can(permission) {
			return false
		}
	}
	return true
}

func normalizePermissions(permissions []string) []string {
	seen := make(map[string]struct{}, len(permissions))
	result := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		permission = normalizePermission(permission)
		if permission == "" {
			continue
		}
		if _, exists := seen[permission]; exists {
			continue
		}
		seen[permission] = struct{}{}
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func normalizePermission(permission string) string {
	return strings.ToLower(strings.TrimSpace(permission))
}
