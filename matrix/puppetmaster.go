package matrix

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/id"
)

type userGhostConfig struct {
	userMXID           id.UserID
	doublePuppetIntent *appservice.IntentAPI
	ghost              *Ghost
}

type GhostMaster struct {
	bridge          *bridge.Bridge
	localpart       string
	userGhostConfig map[id.UserID]*userGhostConfig
}

func NewGhostMaster(bridge *bridge.Bridge, localpart string) *GhostMaster {
	return &GhostMaster{
		bridge:          bridge,
		localpart:       localpart,
		userGhostConfig: make(map[id.UserID]*userGhostConfig),
	}
}

// NewGhost creates a new ghost user with the given remote ID, display name, username and avatar URL.
func (pm *GhostMaster) NewGhost(remoteID string, displayName, userName string, avatarURL id.ContentURI) *Ghost {
	mxid := id.NewUserID(fmt.Sprintf("%s_%s", pm.localpart, userName), pm.bridge.Config.Homeserver.Domain)
	fmt.Println("[CreatePuppet] ", userName, mxid.String())

	return &Ghost{
		MXID:        mxid,
		RemoteID:    remoteID,
		DisplayName: displayName,
		UserName:    userName,
		Intent:      pm.bridge.AS.Intent(mxid),
		AvatarURL:   avatarURL,
	}
}

// LoadGhostIntent loads the intent for the given ghost and fills it into the struct.
// DEPRECATED: use AsGhost instead.
func (pm *GhostMaster) LoadGhostIntent(ghost *Ghost) *Ghost {
	ghost.Intent = pm.bridge.AS.Intent(ghost.MXID)
	return ghost
}

// HasDoublePuppet checks if the user has a doublePuppet intent.
// This will NOT try to setup the double puppet intent if it doesn't already exist yet,
// so even if the user theoretically can double puppet, Setup has to get called first.
func (pm *GhostMaster) HasDoublePuppet(user *User) bool {
	userGhostConfig, ok := pm.userGhostConfig[user.MXID]
	if !ok {
		return false
	}

	return userGhostConfig.doublePuppetIntent != nil
}

// AsGhost returns an intent to impersonate the given ghost
func (pm *GhostMaster) AsGhost(ghost *Ghost) *appservice.IntentAPI {
	intent := pm.bridge.AS.Intent(ghost.MXID)
	fmt.Println("AsGhost: ", ghost.MXID, intent.UserID)
	return intent
}

// AsUserGhost checks if the user has a doublePuppet intent.
// If yes, it returns the intent.
// If not, it tries to setup double puppeting for the user.
// If that fails, it tries to create a ghost for the current user.
func (pm *GhostMaster) AsUserGhost(ctx context.Context, user *User) *appservice.IntentAPI {
	fmt.Println("AsUserGhost: ", user.MXID)
	userGhostConfig, ok := pm.userGhostConfig[user.MXID]
	if !ok {
		fmt.Println("not processed yet")
		pm.SetupDoublePuppet(ctx, user)
		pm.SetupUserGhost(ctx, user)

		userGhostConfig, ok = pm.userGhostConfig[user.MXID]
	}

	if userGhostConfig.doublePuppetIntent != nil {
		return userGhostConfig.doublePuppetIntent
	}

	if userGhostConfig.ghost == nil {
		ghost, _ := pm.SetupUserGhost(ctx, user)
		userGhostConfig.ghost = ghost
	}

	fmt.Println("no double puppet intent, using normal ghost: ", userGhostConfig.ghost)
	return pm.AsGhost(userGhostConfig.ghost)
}

// UploadGhostAvatar uploads a new avatar for the given ghost and returns the new avatar URL as ContentURI.
func (pm *GhostMaster) UploadGhostAvatar(ctx context.Context, ghost *Ghost, url string) (id.ContentURI, error) {
	bot := pm.bridge.AS.BotClient()

	getResp, err := http.DefaultClient.Get(url)
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to download avatar: %w", err)
	}

	data, err := io.ReadAll(getResp.Body)
	_ = getResp.Body.Close()
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to read avatar bytes: %w", err)
	}

	// upload to matrix
	resp, err := bot.UploadBytes(ctx, data, "image/png")
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to upload avatar to Matrix: %w", err)
	}
	fmt.Println("Uploaded image to matrix servers -- ", resp.ContentURI.String())

	fmt.Println("puppet intent: ", ghost.DefaultIntent().UserID)

	err = pm.AsGhost(ghost).SetAvatarURL(ctx, resp.ContentURI)

	return resp.ContentURI, err
}

// UpdateGhostName updates the name of the given ghost.
func (pm *GhostMaster) UpdateGhostName(ctx context.Context, ghost *Ghost, newName string) error {
	ghost.DisplayName = newName
	err := pm.AsGhost(ghost).SetDisplayName(ctx, newName)
	if err != nil {
		fmt.Sprintf("could not update name: ", err.Error())
		return err
	}

	return nil
}

// SetupUserGhost creates a normal ghost for the given user.
// This ghost will be used when the user does not have a double puppet intent.
func (pm *GhostMaster) SetupUserGhost(ctx context.Context, user *User) (*Ghost, error) {
	fmt.Println("SetupUserGhost: ", user.MXID)
	userGhost := pm.NewGhost(user.RemoteID, user.DisplayName, user.MXID.Localpart(), id.ContentURI{})

	conf, ok := pm.userGhostConfig[user.MXID]
	if !ok {
		pm.userGhostConfig[user.MXID] = &userGhostConfig{}
		conf = pm.userGhostConfig[user.MXID]
	}

	conf.ghost = userGhost

	return conf.ghost, nil
}

// SetupDoublePuppet creates a double puppet intent for the given user, if possible.
// Double puppeting needs to be enabled for this to work
func (pm *GhostMaster) SetupDoublePuppet(ctx context.Context, user *User) (*appservice.IntentAPI, error) {
	c := pm.bridge.AS.Client(user.MXID)
	newIntent, newAccessToken, err := pm.bridge.DoublePuppet.Setup(ctx, user.MXID, c.AccessToken, true)
	if err != nil {
		fmt.Println("Error setting up double puppet: ", err)
		return nil, err
	}

	user.AccessToken = newAccessToken
	user.DoublePuppetIntent = newIntent

	conf, ok := pm.userGhostConfig[user.MXID]
	if !ok {
		pm.userGhostConfig[user.MXID] = &userGhostConfig{}
		conf = pm.userGhostConfig[user.MXID]
	}

	conf.userMXID = user.MXID
	conf.doublePuppetIntent = newIntent

	fmt.Println("double puppetting ok")

	return newIntent, nil
}
