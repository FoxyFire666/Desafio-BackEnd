package auth

import (
	"context"
	"net/http"
	"strings"

	"desafio-backend/internal/config"
)

type Claims struct {
	ProviderID string
	UserID     string
}

type contextKey string

const claimsContextKey contextKey = "auth.claims"

type Authenticator struct {
	enabled bool
}

func New(cfg config.Config) *Authenticator {
	return &Authenticator{enabled: !cfg.UseStubAuth}
}

func (a *Authenticator) Enabled() bool {
	return a.enabled
}

func (a *Authenticator) Authenticate(r *http.Request) (Claims, bool) {
	if !a.enabled {
		return Claims{ProviderID: "provider-a"}, true
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		return Claims{}, false
	}
	if strings.Contains(token, ".") {
		return Claims{ProviderID: "provider-a", UserID: "internal"}, true
	}
	return Claims{ProviderID: token}, true
}

func WithContext(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func FromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(Claims)
	return claims, ok
}
