package backup

import (
	"fmt"
	"os/user"
	"strconv"
)

// Credential is a user to run restic as
type Credential struct {
	Uid    uint32
	Gid    uint32
	Groups []uint32
	Home   string
}

// LookupCredential looks up a user's IDs, groups and home directory
func LookupCredential(name string) (*Credential, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return nil, err
	}
	uid, err := parseID(u.Uid)
	if err != nil {
		return nil, err
	}
	gid, err := parseID(u.Gid)
	if err != nil {
		return nil, err
	}
	groupIDs, err := u.GroupIds()
	if err != nil {
		return nil, err
	}
	groups := make([]uint32, 0, len(groupIDs))
	for _, g := range groupIDs {
		id, err := parseID(g)
		if err != nil {
			return nil, err
		}
		groups = append(groups, id)
	}
	return &Credential{Uid: uid, Gid: gid, Groups: groups, Home: u.HomeDir}, nil
}

func parseID(s string) (uint32, error) {
	id, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", s, err)
	}
	return uint32(id), nil
}
