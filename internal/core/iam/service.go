// Package iam owns users, sessions, the org-unit tree and roles.
package iam

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"maps"
	"net/http"
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/iam/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct {
	d          Deps
	roles      map[string]map[string][]string // product → role → permissions
	tenantWide map[[2]string]bool             // product, role
	sensitive  map[[2]string]bool             // product, permission core.admin does not imply
	tree       TreeHook
	data       DataProducts
}

func NewService(d Deps) *Service {
	s := &Service{d: d, roles: map[string]map[string][]string{}, tenantWide: map[[2]string]bool{}, sensitive: map[[2]string]bool{}}
	s.RegisterRoles("core", map[string][]string{"admin": {PermManageOrg, PermManageUsers, PermManagePeriods, PermManageApproval, PermMonitorJobs, PermManageMail, PermManagePrint, setting.PermManage}}, "admin")
	return s
}

// SetTreeHook wires the tree hook while composing the app.
func (s *Service) SetTreeHook(h TreeHook) { s.tree = h }

// SetDataProducts wires the DataProducts hook while composing the app.
func (s *Service) SetDataProducts(d DataProducts) { s.data = d }

// Compared against when the login does not exist, so both failures cost the same time.
var dummyHash = hashPassword("dummy password")

// CreateUser adds a user who signs in with login (case-insensitive) and password.
func (s *Service) CreateUser(ctx context.Context, login, name, password string) (int64, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return 0, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = s.createUser(ctx, login, name, password); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.user_created", map[string]any{"id": id, "login": login})
	})
	return id, err
}

// CreateAdmin adds the first administrator at install time, with the core admin role tenant-wide.
func (s *Service) CreateAdmin(ctx context.Context, login, name, password string) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = s.createUser(ctx, login, name, password); err != nil {
			return err
		}
		if _, err := store.New(platform.DBFrom(ctx)).GrantRole(ctx, store.GrantRoleParams{UserID: id, Product: "core", Role: "admin"}); err != nil {
			return err
		}
		// No actor: the install command acts, not a user.
		return s.d.Audit.Record(ctx, "iam.admin_created", map[string]any{"id": id, "login": login})
	})
	return id, err
}

func (s *Service) createUser(ctx context.Context, login, name, password string) (int64, error) {
	if len(password) < MinPasswordLength {
		return 0, ErrPasswordTooShort
	}
	id, err := store.New(platform.DBFrom(ctx)).CreateUser(ctx, store.CreateUserParams{Login: login, Name: name, PasswordHash: hashPassword(password)})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return 0, ErrLoginTaken
	}
	return id, err
}

// Login checks the password and opens a session; the returned token goes into the cookie.
func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	// Hashing runs outside the transaction so it holds no connection.
	u, err := store.New(platform.DBFrom(ctx)).GetUserByLogin(ctx, login)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		verifyPassword(password, dummyHash)
		return "", s.loginFailed(ctx, login)
	case err != nil:
		return "", err
	case !verifyPassword(password, u.PasswordHash):
		return "", s.loginFailed(ctx, login)
	}
	token := rand.Text()
	err = platform.InTx(platform.WithActor(ctx, u.ID), func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		if err := q.DeleteExpiredSessions(ctx, u.ID); err != nil {
			return err
		}
		if err := q.CreateSession(ctx, store.CreateSessionParams{TokenHash: tokenHash(token), UserID: u.ID}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.login", nil)
	})
	return token, err
}

func (s *Service) loginFailed(ctx context.Context, login string) error {
	if err := s.d.Audit.Record(ctx, "iam.login_failed", map[string]any{"login": login}); err != nil {
		return err
	}
	return ErrInvalidCredentials
}

// Logout ends the session of token; an unknown token is already logged out.
func (s *Service) Logout(ctx context.Context, token string) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		userID, err := store.New(platform.DBFrom(ctx)).DeleteSession(ctx, tokenHash(token))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(platform.WithActor(ctx, userID), "iam.logout", nil)
	})
}

// Authenticate resolves a session token; ok is false when it is unknown or expired.
func (s *Service) Authenticate(ctx context.Context, token string) (sess Session, ok bool, err error) {
	row, err := store.New(platform.DBFrom(ctx)).GetSession(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	return Session{UserID: row.UserID, AuthzVersion: authzVersion(row.AuthzVersion, row.TenantVersion)}, true, nil
}

// Me describes the actor in ctx.
func (s *Service) Me(ctx context.Context) (Me, error) {
	id, _ := platform.ActorFrom(ctx)
	u, err := store.New(platform.DBFrom(ctx)).GetUser(ctx, id)
	if err != nil {
		return Me{}, err
	}
	locale := u.Locale.String
	if !u.Locale.Valid {
		if locale, err = s.d.Setting.Get(ctx, setting.Locale); err != nil {
			return Me{}, err
		}
	}
	tz, err := s.d.Setting.Get(ctx, setting.Timezone)
	if err != nil {
		return Me{}, err
	}
	perms, err := s.permissions(ctx, id)
	if err != nil {
		return Me{}, err
	}
	products := slices.Sorted(slices.Values(platform.ProductsFrom(ctx)))
	if products == nil {
		products = []string{}
	}
	withData := []string{}
	if s.data != nil {
		if withData, err = s.data.ProductsWithData(ctx); err != nil {
			return Me{}, err
		}
	}
	return Me{
		ID: u.ID, Login: u.Login, Name: u.Name, Locale: locale, Timezone: tz,
		AuthzVersion: authzVersion(u.AuthzVersion, u.TenantVersion), Products: products, ProductsWithData: withData, Permissions: perms,
	}, nil
}

// SetLocale saves the actor's language.
func (s *Service) SetLocale(ctx context.Context, locale string) error {
	id, _ := platform.ActorFrom(ctx)
	return store.New(platform.DBFrom(ctx)).SetLocale(ctx, store.SetLocaleParams{ID: id, Locale: pgtype.Text{String: locale, Valid: true}})
}

// Users lists every account.
func (s *Service) Users(ctx context.Context) ([]User, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListUsers(ctx)
	out := make([]User, len(rows))
	for i, r := range rows {
		out[i] = User{ID: r.ID, Login: r.Login, Name: r.Name, Email: platform.TextPtr(r.Email)}
	}
	return out, err
}

// User returns one account.
func (s *Service) User(ctx context.Context, id int64) (User, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return User{}, err
	}
	u, err := store.New(platform.DBFrom(ctx)).GetUser(ctx, id)
	return User{ID: u.ID, Login: u.Login, Name: u.Name, Email: platform.TextPtr(u.Email)}, notFound(err)
}

var errEmail = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"email"}}}

// SetEmail sets where a user's email notifications go; nil or blank removes it. Only a
// bare address is taken, no display name.
func (s *Service) SetEmail(ctx context.Context, id int64, email *string) error {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return err
	}
	if email != nil {
		if *email = strings.TrimSpace(*email); *email == "" {
			email = nil
		} else if a, err := mail.ParseAddress(*email); err != nil || a.Address != *email || len(*email) > 254 {
			return errEmail
		}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		n, err := store.New(platform.DBFrom(ctx)).SetUserEmail(ctx, store.SetUserEmailParams{ID: id, Email: platform.NullText(email)})
		if err != nil {
			return err
		}
		if n == 0 {
			return platform.ErrNotFound
		}
		return s.d.Audit.Record(ctx, "iam.user_email_set", map[string]any{"id": id})
	})
}

// UserIDByLogin resolves a login (case-insensitive) for modules that link records to accounts.
func (s *Service) UserIDByLogin(ctx context.Context, login string) (int64, error) {
	u, err := store.New(platform.DBFrom(ctx)).GetUserByLogin(ctx, login)
	return u.ID, notFound(err)
}

// authzVersion joins the user's and the tenant's counters; either one moving changes it.
func authzVersion(user, tenant int64) string {
	return strconv.FormatInt(user, 10) + "." + strconv.FormatInt(tenant, 10)
}

func tokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// Core roles are tenant-wide.
const (
	PermManageOrg      = "core.org.manage"
	PermManageUsers    = "core.user.manage"
	PermManagePeriods  = "core.period.manage"
	PermManageApproval = "core.approval.manage"
	// PermMonitorJobs lists system jobs, failing ones included.
	PermMonitorJobs = "core.job.monitor"
	// PermManageMail sets the tenant's mail server; the notification module checks it.
	PermManageMail = "core.mail.manage"
	// PermManagePrint edits the text blocks of print templates; the printing module checks it.
	PermManagePrint = "core.print_template.manage"
)

// RegisterRoles declares a product's roles and the permissions each grants;
// tenantWide roles cannot be granted at an org unit. Call it while wiring modules, before serving.
func (s *Service) RegisterRoles(product string, roles map[string][]string, tenantWide ...string) {
	if _, dup := s.roles[product]; dup {
		panic("iam: roles of " + product + " registered twice")
	}
	s.roles[product] = roles
	for _, r := range tenantWide {
		if _, ok := roles[r]; !ok {
			panic("iam: tenant-wide role " + product + "." + r + " not registered")
		}
		s.tenantWide[[2]string{product, r}] = true
	}
}

// RegisterSensitive marks permissions of a product that guard sensitive data: unlike its
// other permissions, a tenant administrator holds them only through a role granted to them.
// Call it while wiring modules, before serving.
func (s *Service) RegisterSensitive(product string, permissions ...string) {
	for _, p := range permissions {
		s.sensitive[[2]string{product, p}] = true
	}
}

// adminImplies reports whether a tenant-wide core.admin holds perm of product without a role.
func (s *Service) adminImplies(product, perm string) bool {
	return product != "core" && !s.sensitive[[2]string{product, perm}]
}

// isAdmin: core.admin granted tenant-wide (it can be granted no other way).
func isAdmin(grants []store.UserRolesRow) bool {
	return slices.ContainsFunc(grants, func(g store.UserRolesRow) bool { return g.Product == "core" && g.Role == "admin" && !g.OrgUnitID.Valid })
}

// Roles lists every registered role.
func (s *Service) Roles() []Role {
	var out []Role
	for _, p := range slices.Sorted(maps.Keys(s.roles)) {
		for _, r := range slices.Sorted(maps.Keys(s.roles[p])) {
			out = append(out, Role{Product: p, Role: r, Permissions: s.roles[p][r], TenantWide: s.tenantWide[[2]string{p, r}]})
		}
	}
	return out
}

type scopeCacheKey struct{}

type scopeCache struct {
	mu sync.Mutex
	m  map[[2]string]Scope
}

// withScopeCache makes Scope compute once per request.
func withScopeCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, scopeCacheKey{}, &scopeCache{m: map[[2]string]Scope{}})
}

// AsUser acts as another user within ctx, with a scope cache of its own, e.g. to ask
// whether a would-be approver may view a document.
func (s *Service) AsUser(ctx context.Context, userID int64) context.Context {
	return withScopeCache(platform.WithActor(ctx, userID))
}

// Scope returns the org units where the actor has permission, subtrees included.
func (s *Service) Scope(ctx context.Context, product, permission string) (Scope, error) {
	c, _ := ctx.Value(scopeCacheKey{}).(*scopeCache)
	if c != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if sc, ok := c.m[[2]string{product, permission}]; ok {
			return sc, nil
		}
	}
	sc, err := s.scope(ctx, product, permission)
	if err == nil && c != nil {
		c.m[[2]string{product, permission}] = sc
	}
	return sc, err
}

func (s *Service) scope(ctx context.Context, product, permission string) (Scope, error) {
	userID, ok := platform.ActorFrom(ctx)
	if !ok {
		return Scope{}, nil
	}
	grants, err := store.New(platform.DBFrom(ctx)).UserRoles(ctx, userID)
	if err != nil {
		return Scope{}, err
	}
	if isAdmin(grants) && s.adminImplies(product, permission) && s.registered(product, permission) {
		return Scope{All: true}, nil
	}
	var roots []int64
	for _, g := range grants {
		if g.Product != product || !slices.Contains(s.roles[product][g.Role], permission) {
			continue
		}
		if !g.OrgUnitID.Valid {
			return Scope{All: true}, nil
		}
		roots = append(roots, g.OrgUnitID.Int64)
	}
	if len(roots) == 0 {
		return Scope{}, nil
	}
	units, err := store.New(platform.DBFrom(ctx)).Subtrees(ctx, roots)
	return Scope{Units: units}, err
}

// registered: some role of product grants permission, so a mistyped name never opens.
func (s *Service) registered(product, permission string) bool {
	for _, perms := range s.roles[product] {
		if slices.Contains(perms, permission) {
			return true
		}
	}
	return false
}

// Allowed reports whether the actor holds product's permission at unit and at each of more.
func (s *Service) Allowed(ctx context.Context, product, permission string, unit int64, more ...int64) (bool, error) {
	sc, err := s.Scope(ctx, product, permission)
	if err != nil {
		return false, err
	}
	return sc.Has(unit) && !slices.ContainsFunc(more, func(u int64) bool { return !sc.Has(u) }), nil
}

// Require fails with forbidden unless Allowed.
func (s *Service) Require(ctx context.Context, product, permission string, unit int64, more ...int64) error {
	ok, err := s.Allowed(ctx, product, permission, unit, more...)
	if err == nil && !ok {
		return platform.ErrForbidden
	}
	return err
}

// RequireTenantWide fails with forbidden unless the actor holds product's permission tenant-wide.
func (s *Service) RequireTenantWide(ctx context.Context, product, permission string) error {
	sc, err := s.Scope(ctx, product, permission)
	if err == nil && !sc.All {
		return platform.ErrForbidden
	}
	return err
}

// RequireCore fails with forbidden unless the actor has a tenant-wide core permission.
func (s *Service) RequireCore(ctx context.Context, permission string) error {
	return s.require(ctx, permission)
}

func (s *Service) require(ctx context.Context, permission string) error {
	return s.RequireTenantWide(ctx, "core", permission)
}

// permissions lists every permission the user has anywhere, by product.
func (s *Service) permissions(ctx context.Context, userID int64) (map[string][]string, error) {
	grants, err := store.New(platform.DBFrom(ctx)).UserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	if isAdmin(grants) {
		for product, roles := range s.roles {
			for _, perms := range roles {
				for _, p := range perms {
					if s.adminImplies(product, p) && !slices.Contains(out[product], p) {
						out[product] = append(out[product], p)
					}
				}
			}
		}
	}
	for _, g := range grants {
		for _, p := range s.roles[g.Product][g.Role] {
			if !slices.Contains(out[g.Product], p) {
				out[g.Product] = append(out[g.Product], p)
			}
		}
	}
	for _, ps := range out {
		slices.Sort(ps)
	}
	return out, nil
}

// UserRoles lists the roles granted to a user.
func (s *Service) UserRoles(ctx context.Context, userID int64) ([]Grant, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).UserRoles(ctx, userID)
	out := make([]Grant, len(rows))
	for i, r := range rows {
		out[i] = Grant{ID: r.ID, Product: r.Product, Role: r.Role, OrgUnitID: platform.Int8Ptr(r.OrgUnitID), OrgUnitName: platform.TextPtr(r.OrgUnitName)}
	}
	return out, err
}

// GrantRole gives a user a role at an org unit, or tenant-wide when unit is nil.
func (s *Service) GrantRole(ctx context.Context, userID int64, product, role string, unit *int64) (int64, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return 0, err
	}
	if _, ok := s.roles[product][role]; !ok {
		return 0, ErrUnknownRole
	}
	if s.tenantWide[[2]string{product, role}] && unit != nil {
		return 0, ErrRoleTenantWide
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		var err error
		id, err = q.GrantRole(ctx, store.GrantRoleParams{UserID: userID, Product: product, Role: role, OrgUnitID: platform.NullInt8(unit)})
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23505":
				return ErrRoleAlreadyGranted
			case "23503":
				return platform.ErrNotFound
			}
		}
		if err != nil {
			return err
		}
		if err := q.BumpUserAuthz(ctx, userID); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.role_granted", map[string]any{"user_id": userID, "product": product, "role": role, "org_unit_id": unit})
	})
	return id, err
}

// RevokeRole removes one grant of a user.
func (s *Service) RevokeRole(ctx context.Context, userID, grantID int64) error {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		g, err := q.RevokeRole(ctx, store.RevokeRoleParams{ID: grantID, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		if err := q.BumpUserAuthz(ctx, userID); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.role_revoked", map[string]any{"user_id": userID, "product": g.Product, "role": g.Role, "org_unit_id": platform.Int8Ptr(g.OrgUnitID)})
	})
}

// OrgUnits lists the whole tree for any signed-in user, or with permission set,
// only the units where the actor has that permission.
func (s *Service) OrgUnits(ctx context.Context, product, permission string) ([]OrgUnit, error) {
	var sc Scope
	var err error
	if permission == "" {
		if _, ok := platform.ActorFrom(ctx); !ok {
			return nil, platform.ErrForbidden
		}
		sc.All = true
	} else {
		sc, err = s.Scope(ctx, product, permission)
	}
	if err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListOrgUnits(ctx)
	out := []OrgUnit{}
	for _, r := range rows {
		if sc.Has(r.ID) {
			out = append(out, orgUnit(store.IamOrgUnit(r)))
		}
	}
	return out, err
}

// CreateOrgUnit adds a node to the tree.
func (s *Service) CreateOrgUnit(ctx context.Context, in OrgUnitInput) (int64, error) {
	if err := s.require(ctx, PermManageOrg); err != nil {
		return 0, err
	}
	if err := in.check(); err != nil {
		return 0, err
	}
	var id int64
	err := s.writeTree(ctx, func(ctx context.Context) error {
		var err error
		id, err = store.New(platform.DBFrom(ctx)).CreateOrgUnit(ctx, store.CreateOrgUnitParams{
			ParentID: platform.NullInt8(in.ParentID), Kind: in.Kind, Name: in.Name,
			TaxCode: platform.NullText(in.TaxCode), LegalName: platform.NullText(in.LegalName), Address: platform.NullText(in.Address),
		})
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.org_unit_created", map[string]any{"id": id, "parent_id": in.ParentID, "kind": in.Kind, "name": in.Name})
	})
	return id, err
}

// UpdateOrgUnit edits or moves a node. A move changes subtree permissions at once.
func (s *Service) UpdateOrgUnit(ctx context.Context, id int64, in OrgUnitInput) error {
	if err := s.require(ctx, PermManageOrg); err != nil {
		return err
	}
	if err := in.check(); err != nil {
		return err
	}
	return s.writeTree(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		old, err := q.GetOrgUnit(ctx, id)
		if err != nil {
			return notFound(err)
		}
		if _, err := q.UpdateOrgUnit(ctx, store.UpdateOrgUnitParams{
			ID: id, ParentID: platform.NullInt8(in.ParentID), Kind: in.Kind, Name: in.Name,
			TaxCode: platform.NullText(in.TaxCode), LegalName: platform.NullText(in.LegalName), Address: platform.NullText(in.Address),
		}); err != nil {
			return err
		}
		if old.ParentID != platform.NullInt8(in.ParentID) {
			if err := q.BumpTenantAuthz(ctx); err != nil {
				return err
			}
		}
		return s.d.Audit.Record(ctx, "iam.org_unit_updated", map[string]any{
			"id": id, "parent_id": in.ParentID, "old_parent_id": platform.Int8Ptr(old.ParentID), "kind": in.Kind, "name": in.Name,
		})
	})
}

// writeTree runs fn in one transaction under the tree lock, then checks the tree's shape.
// fn must use the ctx it is given, so its writes and audit entry roll back together.
func (s *Service) writeTree(ctx context.Context, fn func(ctx context.Context) error) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		if err := q.LockOrgUnits(ctx); err != nil {
			return err
		}
		if err := fn(ctx); err != nil {
			if isFKViolation(err) {
				return platform.ErrNotFound
			}
			return err
		}
		v, err := q.OrgTreeViolations(ctx)
		switch {
		case err != nil:
			return err
		case v.Unreachable > 0:
			return ErrOrgUnitCycle
		case v.Misplaced > 0:
			return ErrOrgUnitMisplaced
		}
		if s.tree != nil {
			return s.tree.TreeChanged(ctx)
		}
		return nil
	})
}

// ShareTree holds tree writes off until ctx's transaction ends, so a legal entity read
// after it stays the one a unit belongs to (a document never changes legal entity).
func (s *Service) ShareTree(ctx context.Context) error {
	return store.New(platform.DBFrom(ctx)).ShareOrgUnits(ctx)
}

// LegalEntityOf returns the legal entity a unit belongs to: the nearest company at or above it.
func (s *Service) LegalEntityOf(ctx context.Context, unit int64) (int64, error) {
	id, err := store.New(platform.DBFrom(ctx)).LegalEntityOf(ctx, unit)
	return id, notFound(err)
}

// UsersWithRole lists users holding a role that covers unit: granted there, above it, or tenant-wide.
func (s *Service) UsersWithRole(ctx context.Context, product, role string, unit int64) ([]int64, error) {
	return store.New(platform.DBFrom(ctx)).UsersWithRole(ctx, store.UsersWithRoleParams{Unit: unit, Product: product, Role: role})
}

func (in OrgUnitInput) check() error {
	if in.Kind != "company" && (in.TaxCode != nil || in.LegalName != nil || in.Address != nil) {
		return ErrLegalFieldsCompanyOnly
	}
	return nil
}

func isFKViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == "23503"
}

func orgUnit(r store.IamOrgUnit) OrgUnit {
	return OrgUnit{ID: r.ID, OrgUnitInput: OrgUnitInput{ParentID: platform.Int8Ptr(r.ParentID), Kind: r.Kind, Name: r.Name,
		TaxCode: platform.TextPtr(r.TaxCode), LegalName: platform.TextPtr(r.LegalName), Address: platform.TextPtr(r.Address)}}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ErrNotFound
	}
	return err
}
