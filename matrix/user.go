package matrix

import (
	"errors"
	"fmt"

	"maunium.net/go/mautrix/appservice"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/id"
)

var _ bridge.User = &User{}
var _ bridge.DoublePuppet = (*User)(nil)

type SetManagementRoomHandler func(*User, id.RoomID)

type User struct {
	// ID is the ID of the user in the Matrix homeserver.
	MXID               id.UserID                    `json:"mxid,omitempty"`
	RemoteID           string                       `json:"remote_id,omitempty"`
	RemoteName         string                       `json:"remote_name,omitempty"`
	DisplayName        string                       `json:"display_name,omitempty"`
	PermissionLevel    bridgeconfig.PermissionLevel `json:"permission_level,omitempty"`
	ManagementRoomID   id.RoomID                    `json:"management_room_id,omitempty"`
	BridgeState        *bridge.BridgeStateQueue     `json:"-"`
	DoublePuppetIntent *appservice.IntentAPI        `json:"-"`
	AccessToken        string                       `json:"access_token,omitempty"`

	SetManagementRoomHandler SetManagementRoomHandler `json:"-"`
}

// -- double puppet
func (u *User) CustomIntent() *appservice.IntentAPI {
	return u.DoublePuppetIntent
}

func (u *User) SwitchCustomMXID(accessToken string, userID id.UserID) error {
	fmt.Println("[SwitchCustomMXID] ", userID, accessToken)
	if userID != u.MXID {
		return errors.New("mismatching mxid")
	}

	u.DoublePuppetIntent = nil
	u.AccessToken = accessToken

	return nil
}

func (u *User) ClearCustomMXID() {
	u.DoublePuppetIntent = nil
	u.AccessToken = ""
}

// -- end double puppet

// GetIDoublePuppet implements bridge.User.
func (u *User) GetIDoublePuppet() bridge.DoublePuppet {
	panic("GetIDoublePuppet unimplemented")
}

// GetIGhost implements bridge.User.
func (u *User) GetIGhost() bridge.Ghost {
	panic("GetIGhost " + u.DisplayName + "  unimplemented")
	return nil
}

// GetMXID implements bridge.User.
func (u *User) GetMXID() id.UserID {
	return u.MXID
}

// GetManagementRoomID implements bridge.User.
func (u *User) GetManagementRoomID() id.RoomID {
	fmt.Println("[GetManagementRoomID] ", u.ManagementRoomID)
	return u.ManagementRoomID
}

// GetPermissionLevel implements bridge.User.
func (u *User) GetPermissionLevel() bridgeconfig.PermissionLevel {
	fmt.Println("[GetPermissionLevel] ", u.PermissionLevel)
	return u.PermissionLevel
}

// IsLoggedIn implements bridge.User.
func (u *User) IsLoggedIn() bool {
	fmt.Println("IsLoggedIn unimplemented")
	return true
}

// SetManagementRoom implements bridge.User.
func (u *User) SetManagementRoom(rid id.RoomID) {
	fmt.Println("[SetManagementRoom] ", rid.String())
	u.SetManagementRoomHandler(u, rid)
	u.ManagementRoomID = rid
}

// for bridge state

func (u *User) GetRemoteID() string {
	fmt.Println("[GetRemoteID] ", u.RemoteID)
	return u.RemoteID
}

func (u *User) GetRemoteName() string {
	fmt.Println("[GetRemoteName] ", u.RemoteName)
	return u.RemoteName
}
