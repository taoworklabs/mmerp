package iam

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const cookieName = "session"

type loginInput struct {
	Body struct {
		Login    string `json:"login" minLength:"1" maxLength:"200"`
		Password string `json:"password" minLength:"1" maxLength:"200"`
	}
}

type sessionInput struct {
	Session string `cookie:"session"`
}

type cookieOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type meOutput struct{ Body Me }

type updateMeInput struct {
	Body struct {
		Locale string `json:"locale" enum:"vi,en"`
	}
}

// maxAge 0 makes a browser-session cookie (the server expiry still applies); -1 deletes it.
func sessionCookie(token string, maxAge int) http.Cookie {
	return http.Cookie{Name: cookieName, Value: token, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
}

func registerHandlers(api huma.API, s *Service) {
	public := map[string]any{platform.MetaPublic: true}

	huma.Register(api, huma.Operation{
		OperationID: "login", Method: http.MethodPost, Path: "/auth/login", Metadata: public,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *loginInput) (*cookieOutput, error) {
		token, err := s.Login(ctx, in.Body.Login, in.Body.Password)
		if err != nil {
			return nil, err
		}
		return &cookieOutput{SetCookie: sessionCookie(token, 0)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout", Method: http.MethodPost, Path: "/auth/logout", Metadata: public,
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *sessionInput) (*cookieOutput, error) {
		if in.Session != "" {
			if err := s.Logout(ctx, in.Session); err != nil {
				return nil, err
			}
		}
		return &cookieOutput{SetCookie: sessionCookie("", -1)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-me", Method: http.MethodGet, Path: "/me",
	}, func(ctx context.Context, _ *struct{}) (*meOutput, error) {
		me, err := s.Me(ctx)
		if err != nil {
			return nil, err
		}
		return &meOutput{Body: me}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-me", Method: http.MethodPatch, Path: "/me",
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *updateMeInput) (*struct{}, error) {
		return nil, s.SetLocale(ctx, in.Body.Locale)
	})

	registerAdminHandlers(api, s)
}

type orgUnitsInput struct {
	Product    string `query:"product" doc:"With permission: only units where the actor has it"`
	Permission string `query:"permission"`
}

type orgUnitsOutput struct{ Body []OrgUnit }

type orgUnitInput struct{ Body OrgUnitInput }

type idInput struct {
	ID int64 `path:"id"`
}

type emailInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Email *string `json:"email" maxLength:"254" doc:"Null or blank removes it"`
	}
}

type updateOrgUnitInput struct {
	ID   int64 `path:"id"`
	Body OrgUnitInput
}

type createdOutput struct {
	Body struct {
		ID int64 `json:"id"`
	}
}

type usersOutput struct{ Body []User }

type userOutput struct{ Body User }

type createUserInput struct {
	Body struct {
		Login    string `json:"login" minLength:"1" maxLength:"200"`
		Name     string `json:"name" minLength:"1" maxLength:"200"`
		Password string `json:"password" minLength:"1" maxLength:"200"`
	}
}

type rolesOutput struct{ Body []Role }

type grantsOutput struct{ Body []Grant }

type grantInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Product   string `json:"product"`
		Role      string `json:"role"`
		OrgUnitID *int64 `json:"org_unit_id"`
	}
}

type revokeInput struct {
	ID      int64 `path:"id"`
	GrantID int64 `path:"grant"`
}

func created(id int64, err error) (*createdOutput, error) {
	if err != nil {
		return nil, err
	}
	out := &createdOutput{}
	out.Body.ID = id
	return out, nil
}

func registerAdminHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-org-units", Method: http.MethodGet, Path: "/org-units"},
		func(ctx context.Context, in *orgUnitsInput) (*orgUnitsOutput, error) {
			units, err := s.OrgUnits(ctx, in.Product, in.Permission)
			return &orgUnitsOutput{Body: units}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-org-unit", Method: http.MethodPost, Path: "/org-units", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *orgUnitInput) (*createdOutput, error) {
			return created(s.CreateOrgUnit(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-org-unit", Method: http.MethodPut, Path: "/org-units/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateOrgUnitInput) (*struct{}, error) {
			return nil, s.UpdateOrgUnit(ctx, in.ID, in.Body)
		})

	huma.Register(api, huma.Operation{OperationID: "list-users", Method: http.MethodGet, Path: "/users"},
		func(ctx context.Context, _ *struct{}) (*usersOutput, error) {
			users, err := s.Users(ctx)
			return &usersOutput{Body: users}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-user", Method: http.MethodPost, Path: "/users", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *createUserInput) (*createdOutput, error) {
			return created(s.CreateUser(ctx, in.Body.Login, in.Body.Name, in.Body.Password))
		})
	huma.Register(api, huma.Operation{OperationID: "get-user", Method: http.MethodGet, Path: "/users/{id}"},
		func(ctx context.Context, in *idInput) (*userOutput, error) {
			u, err := s.User(ctx, in.ID)
			return &userOutput{Body: u}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-user-email", Method: http.MethodPut, Path: "/users/{id}/email", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *emailInput) (*struct{}, error) {
			return nil, s.SetEmail(ctx, in.ID, in.Body.Email)
		})
	huma.Register(api, huma.Operation{OperationID: "list-roles", Method: http.MethodGet, Path: "/roles"},
		func(ctx context.Context, _ *struct{}) (*rolesOutput, error) {
			return &rolesOutput{Body: s.Roles()}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "list-user-roles", Method: http.MethodGet, Path: "/users/{id}/roles"},
		func(ctx context.Context, in *idInput) (*grantsOutput, error) {
			grants, err := s.UserRoles(ctx, in.ID)
			return &grantsOutput{Body: grants}, err
		})
	huma.Register(api, huma.Operation{OperationID: "grant-role", Method: http.MethodPost, Path: "/users/{id}/roles", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *grantInput) (*createdOutput, error) {
			return created(s.GrantRole(ctx, in.ID, in.Body.Product, in.Body.Role, in.Body.OrgUnitID))
		})
	huma.Register(api, huma.Operation{OperationID: "revoke-role", Method: http.MethodDelete, Path: "/users/{id}/roles/{grant}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *revokeInput) (*struct{}, error) {
			return nil, s.RevokeRole(ctx, in.ID, in.GrantID)
		})
}

// authenticate resolves the session cookie, if any, into the actor and sets
// X-Authz-Version. Rejecting anonymous requests is left to the API.
func authenticate(s *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			sess, ok, err := s.Authenticate(r.Context(), c.Value)
			if err != nil {
				platform.LogFrom(r.Context()).Error("authenticate", "err", err)
				platform.WriteError(w, &platform.Error{Status: http.StatusInternalServerError, Code: "internal_error"})
				return
			}
			if ok {
				w.Header().Set("X-Authz-Version", sess.AuthzVersion)
				r = r.WithContext(withScopeCache(platform.WithActor(r.Context(), sess.UserID)))
			}
			next.ServeHTTP(w, r)
		})
	}
}
