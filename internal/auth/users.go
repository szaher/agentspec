package auth

// User represents an authenticated user with agent access control.
type User struct {
	Name   string
	Agents []string // allowed agent names (empty = all agents)
	Role   string   // "invoke" or "admin"
}

// UserStore maps API keys to user identities.
type UserStore struct {
	users map[string]*User // key: API key value, val: user
}

// NewUserStore creates a UserStore from a list of user definitions.
// keyValues maps user name → resolved API key value (already resolved from secrets).
func NewUserStore(defs []UserDef) *UserStore {
	store := &UserStore{users: make(map[string]*User)}
	for _, d := range defs {
		if d.ResolvedKey != "" {
			store.users[d.ResolvedKey] = &User{
				Name:   d.Name,
				Agents: d.Agents,
				Role:   d.Role,
			}
		}
	}
	return store
}

// UserDef holds a user definition with the resolved secret key value.
type UserDef struct {
	Name        string
	KeyRef      string // secret reference name (e.g., "ALICE_API_KEY")
	Agents      []string
	Role        string
	ResolvedKey string // actual API key value (resolved from environment)
}

// Resolve looks up a user by API key.
func (s *UserStore) Resolve(apiKey string) (*User, bool) {
	u, ok := s.users[apiKey]
	return u, ok
}

// IsAuthorized checks if the user is allowed to access the given agent.
func (u *User) IsAuthorized(agentName string) bool {
	if u.Role == "admin" {
		return true
	}
	if len(u.Agents) == 0 {
		return true // no restriction = all agents
	}
	for _, a := range u.Agents {
		if a == agentName {
			return true
		}
	}
	return false
}
