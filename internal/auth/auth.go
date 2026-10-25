package auth

import (
	"context"
	"errors"
)

var ErrInvalidIdentity = errors.New("invalid authentication identity")

type Identity struct {
	UserID string
}

type identityKey struct{}

// WithIdentity сохраняет подтверждённого пользователя в контексте запроса.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// ValidIdentity проверяет, что идентификатор пригоден для передачи в gRPC metadata.
func ValidIdentity(identity Identity) bool {
	if identity.UserID == "" {
		return false
	}
	for _, char := range identity.UserID {
		if char < '!' || char > '~' {
			return false
		}
	}
	return true
}
