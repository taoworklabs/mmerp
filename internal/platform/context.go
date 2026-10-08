package platform

import "context"

type actorKey struct{}

// WithActor records the user acting in ctx; audit and permission checks read it.
func WithActor(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, actorKey{}, userID)
}

// ActorFrom returns the acting user, or false for an anonymous request.
func ActorFrom(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(actorKey{}).(int64)
	return id, ok
}
