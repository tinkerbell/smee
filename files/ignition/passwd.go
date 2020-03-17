package ignition

type Group struct {
	Name         string `json:"group"`
	GID          int    `json:"gid,omitempty"`
	PasswordHash string `json:"passwordHash,omitempty"`
}

type Passwd struct {
	Users  []*User  `json:"users,omitempty"`
	Groups []*Group `json:"groups,omitempty"`
}

type User struct {
	Name           string       `json:"name"`
	PasswordHash   string       `json:"passwordHash,omitempty"`
	AuthorizedKeys []string     `json:"sshAuthorizedKeys,omitempty"`
	Create         *UserOptions `json:"create,omitempty"`
}

type UserOptions struct {
	UID          int      `json:"uid,omitempty"`
	GECOS        string   `json:"gecos,omitempty"`
	HomeDir      string   `json:"homeDir,omitempty"`
	NoCreateHome bool     `json:"noCreateHome,omitempty"`
	PrimaryGroup string   `json:"primaryGroup,omitempty"`
	Groups       []string `json:"groups,omitempty"`
	NoUserGroup  bool     `json:"noUserGroup,omitempty"`
	NoLogInit    bool     `json:"noLogInit,omitempty"`
	Shell        string   `json:"shell,omitempty"`
}
