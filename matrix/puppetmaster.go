package matrix

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/id"
)

type GhostMaster struct {
	bridge    *bridge.Bridge
	localpart string
}

func NewGhostMaster(bridge *bridge.Bridge, localpart string) *GhostMaster {
	return &GhostMaster{
		bridge:    bridge,
		localpart: localpart,
	}
}

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

func (pm *GhostMaster) LoadGhostIntent(ghost *Ghost) *Ghost {
	ghost.Intent = pm.bridge.AS.Intent(ghost.MXID)
	return ghost
}

func (pm *GhostMaster) UpdateGhostAvatar(ctx context.Context, puppet *Ghost, url string) (id.ContentURI, error) {
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

	err = puppet.DefaultIntent().SetAvatarURL(ctx, resp.ContentURI)

	return resp.ContentURI, err
}

func (pm *GhostMaster) UpdateName(ctx context.Context, ghost *Ghost, newName string) bool {

	//if ghost.DisplayName == newName {
	//	return false
	//}

	ghost.DisplayName = newName
	err := ghost.DefaultIntent().SetDisplayName(ctx, newName)
	if err != nil {
		fmt.Sprintf("could not update name: ", err.Error())
		return false
	}

	return true
}
