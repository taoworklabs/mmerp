package setting

import "context"

// Authz checks core permissions; iam implements it, but iam itself reads settings.
type Authz interface {
	RequireCore(ctx context.Context, permission string) error
}
