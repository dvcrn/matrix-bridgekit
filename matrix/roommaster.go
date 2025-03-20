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
	bridge           *bridge.Bridge
	ghostMaster      *GhostMaster
	roomEventHandler RoomEventHandler
}

func NewRoomManager(bridge *bridge.Bridge, gm *GhostMaster, roomEventHandler RoomEventHandler) *RoomManager {
	return &RoomManager{
		bridge:           bridge,
		ghostMaster:      gm,
		roomEventHandler: roomEventHandler,
	}
}

func (rm *RoomManager) NewRoom(name string, topic string, ghosts ...*Ghost) *Room {
	return &Room{
		RemotedID:   "",
		MXID:        "",
		Name:        name,
		Topic:       topic,
		Encrypted:   rm.bridge.Config.Bridge.GetEncryptionConfig().Allow && rm.bridge.Config.Bridge.GetEncryptionConfig().Default,
		PrivateChat: len(ghosts) == 1,
		BotIntent:   rm.bridge.Bot,
		Ghosts:      ghosts,

		roomEventHandler: rm.roomEventHandler,
	}
}

// CreatePersonalSpace creates a new private room as "space" for the given user with the specified name and topic.
// If the user is successfully added to the room, the created room response is returned. Otherwise, an error is returned.
func (rm *RoomManager) CreatePersonalSpace(ctx context.Context, user *User, name string, topic string) (*mautrix.RespCreateRoom, error) {
	resp, err := rm.bridge.Bot.CreateRoom(ctx, &mautrix.ReqCreateRoom{
		Visibility: "private",
		Name:       name,
		Topic:      topic,
		InitialState: []*event.Event{{
			Type: event.StateRoomAvatar,
			Content: event.Content{
				Parsed: &event.RoomAvatarEventContent{
					URL: rm.bridge.Config.AppService.Bot.ParsedAvatar.CUString(),
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

// AddRoomToUserSpace adds a room to the user's space, effectively making it a child of the space
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

// AddUserToRoom adds a user to the specified room. If the user has a double puppet intent,
// it ensures the double puppet is joined to the room.
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

// SetRoomName updates the room name
func (rm *RoomManager) SetRoomName(ctx context.Context, room *Room, intent *appservice.IntentAPI, name string) error {
	_, err := intent.SetRoomName(ctx, room.MXID, name)
	return err
}

// SetRoomAvatar sets the avatar for the given room using the provided intent API and content URI.
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

// UploadRoomAvatar downloads the avatar from the provided URL, uploads it to Matrix, and sets it as the room's avatar.
// Returns the Matrix content URI of the uploaded avatar.
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

// LoadRoom loads the bot intent for the given room and loads the ghost intents for any ghosts in the room.
// This method is used when a room is freshly created and has no intent information attached yet
func (rm *RoomManager) LoadRoom(room *Room) {
	if room.BotIntent == nil {
		room.BotIntent = rm.bridge.Bot
	}

	for _, ghost := range room.Ghosts {
		rm.ghostMaster.LoadGhost(ghost)
	}

	if room.roomEventHandler == nil {
		room.roomEventHandler = rm.roomEventHandler
	}
}

func (rm *RoomManager) EncryptRoom(ctx context.Context, room *Room) {
	content := &event.EncryptionEventContent{Algorithm: "m.megolm.v1.aes-sha2"}
	rm.bridge.Bot.SendStateEvent(ctx, room.MXID, event.StateEncryption, "", content)
}
