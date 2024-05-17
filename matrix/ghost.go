package matrix

import (
	"fmt"

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

	Intent *appservice.IntentAPI `json:"-"`
}

func (g *Ghost) GetDisplayname() string {
	fmt.Println("Displayname: ", g.DisplayName)
	return g.DisplayName
}

func (g *Ghost) GetAvatarURL() id.ContentURI {
	return g.AvatarURL
}

func (g *Ghost) CustomIntent() *appservice.IntentAPI {
	//TODO implement me
	panic("[CustomIntent] implement me")
}

func (g *Ghost) SwitchCustomMXID(accessToken string, userID id.UserID) error {
	//TODO implement me
	panic("[SwitchCustomMXID] implement me")
}

func (g *Ghost) ClearCustomMXID() {
	//TODO implement me
	panic("[ClearCustomMXID] implement me")
}

func (g *Ghost) DefaultIntent() *appservice.IntentAPI {
	return g.Intent
	//TODO implement me
	//panic("implement me")
}

func (g *Ghost) GetMXID() id.UserID {
	return g.MXID
	//TODO implement me
	//panic("implement me")
}
