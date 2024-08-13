package matrix

import (
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/id"
)

var _ bridge.Ghost = (*Ghost)(nil)
var _ bridge.GhostWithProfile = (*Ghost)(nil)

type Ghost struct {
	MXID        id.UserID     `json:"mxid,omitempty"`
	RemoteID    string        `json:"remote_id,omitempty"`
	DisplayName string        `json:"display_name,omitempty"`
	UserName    string        `json:"user_name,omitempty"`
	AvatarURL   id.ContentURI `json:"avatar_url,omitempty"`

	ghostMaster *GhostMaster `json:"-"`
}

func (g *Ghost) GetDisplayname() string {
	return g.DisplayName
}

func (g *Ghost) GetAvatarURL() id.ContentURI {
	return g.AvatarURL
}

func (g *Ghost) CustomIntent() *appservice.IntentAPI {
	panic("[CustomIntent] implement me")
}

func (g *Ghost) SwitchCustomMXID(accessToken string, userID id.UserID) error {
	panic("[SwitchCustomMXID] implement me")
}

func (g *Ghost) ClearCustomMXID() {
	panic("[ClearCustomMXID] implement me")
}

// DefaultIntent gets the intent to act as this ghost
// Deprecated: use PuppetMaster.As
func (g *Ghost) DefaultIntent() *appservice.IntentAPI {
	if g.ghostMaster == nil {
		return nil
	}

	return g.ghostMaster.AsGhost(g)
}

func (g *Ghost) GetMXID() id.UserID {
	return g.MXID
}
