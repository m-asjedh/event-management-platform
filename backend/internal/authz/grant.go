package authz

const (
	EventRead     = "event.read"
	MemberRead    = "member.read"
	UserEmailRead = "user.email.read"
	SessionRead   = "session.read"
	SessionCreate = "session.create"
	SessionUpdate = "session.update"
	RoomRead      = "room.read"
	RoomManage    = "room.manage"
)

// Grant is one person's permissions on one event. It is loaded from
// event_members + role_permissions. Handlers ask Can; they do not switch on Role.
type Grant struct {
	Role string
	perm map[string]struct{}
}

func NewGrant(role string, permissions []string) Grant {
	g := Grant{
		Role: role,
		perm: make(map[string]struct{}, len(permissions)),
	}
	for _, p := range permissions {
		g.perm[p] = struct{}{}
	}
	return g
}

func (g Grant) Can(permission string) bool {
	_, ok := g.perm[permission]
	return ok
}
