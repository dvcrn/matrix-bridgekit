package matrix

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"maunium.net/go/mautrix/appservice"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type RoomManager struct {
	bridge      *bridge.Bridge
	ghostMaster *GhostMaster
}

func NewRoomManager(bridge *bridge.Bridge, gm *GhostMaster) *RoomManager {
	return &RoomManager{
		bridge:      bridge,
		ghostMaster: gm,
	}
}

func (rm *RoomManager) NewRoom(name string, topic string, ghosts ...*Ghost) *Room {
	return NewRoom(name, topic, rm.bridge.Bot, ghosts...)
}

func (rm *RoomManager) CreatePersonalSpace(ctx context.Context, user *User, name string, topic string) (*mautrix.RespCreateRoom, error) {
	resp, err := rm.bridge.Bot.CreateRoom(ctx, &mautrix.ReqCreateRoom{
		Visibility: "private",
		Name:       name,
		Topic:      topic,
		InitialState: []*event.Event{{
			Type: event.StateRoomAvatar,
			Content: event.Content{
				Parsed: &event.RoomAvatarEventContent{
					URL: rm.bridge.Config.AppService.Bot.ParsedAvatar,
				},
			},
		}},
		CreationContent: map[string]interface{}{
			"type": event.RoomTypeSpace,
		},
		BeeperAutoJoinInvites: true,
		PowerLevelOverride: &event.PowerLevelsEventContent{
			Users: map[id.UserID]int{
				rm.bridge.Bot.UserID: 9001,
				user.MXID:            50,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if err := rm.AddUserToRoom(ctx, resp.RoomID, user); err != nil {
		fmt.Println("Err adding user to room: ", err)
	}

	return resp, nil
}

func (rm *RoomManager) AddRoomToUserSpace(ctx context.Context, spaceID id.RoomID, room *Room) error {
	fmt.Println("[AddRoomToUserSpace] ", room.Name)
	_, err := rm.bridge.Bot.SendStateEvent(ctx, spaceID, event.StateSpaceChild, room.MXID.String(), &event.SpaceChildEventContent{
		Via: []string{rm.bridge.Config.Homeserver.Domain},
	})
	if err != nil {
		return err
	}

	return nil
}

func (rm *RoomManager) AddUserToRoom(ctx context.Context, roomID id.RoomID, user *User) error {
	if _, err := rm.bridge.Bot.InviteUser(ctx, roomID, &mautrix.ReqInviteUser{UserID: user.MXID}); err != nil {
		fmt.Println("could not InviteUser to SpaceRoom :of ", err.Error())
		return err
	}

	if user.DoublePuppetIntent != nil {
		err := user.DoublePuppetIntent.EnsureJoined(ctx, roomID, appservice.EnsureJoinedParams{IgnoreCache: true})
		if err != nil {
			fmt.Println("could not call 'ensurejoined' ", err.Error())
		}

		return err
	}

	return nil
}

func (rm *RoomManager) SetRoomAvatar(ctx context.Context, room *Room, intent *appservice.IntentAPI, url id.ContentURI) error {
	_, err := intent.SetRoomAvatar(ctx, room.MXID, url)
	return err
}

func (rm *RoomManager) InsertSetRoomAvatarEvent(ctx context.Context, room *Room, intent *appservice.IntentAPI, url id.ContentURI, ts int64) error {
	_, err := intent.SendMassagedStateEvent(ctx, room.MXID, event.StateRoomAvatar, "", map[string]interface{}{
		"url": url.String(),
	}, ts)

	return err
}

func (rm *RoomManager) UploadRoomAvatar(ctx context.Context, room *Room, intent *appservice.IntentAPI, url string) (id.ContentURI, error) {
	bot := rm.bridge.AS.BotClient()

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

	err = rm.SetRoomAvatar(ctx, room, intent, resp.ContentURI)

	return resp.ContentURI, err
}

func (rm *RoomManager) LoadRoomIntent(room *Room) {
	if room.BotIntent == nil {
		room.BotIntent = rm.bridge.Bot
	}
	for _, ghost := range room.Ghosts {
		rm.ghostMaster.LoadGhostIntent(ghost)
	}
}
