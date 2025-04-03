package bridgekit

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dvcrn/matrix-bridgekit/matrix"
	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/appservice"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/bridge/commands"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// Information to find out exactly which commit the bridge was built from.
// These are filled at build time with the -X linker flag.
var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type ConfigGetter interface {
	DoUpgrade(configupgrade.Helper)
	GetPtr(*bridgeconfig.BaseConfig) any
	Bridge() bridgeconfig.BridgeConfig
}

type BridgeKit[T ConfigGetter] struct {
	bridge.Bridge
	localpart     string
	Config        T
	exampleConfig string
	Commands      []commands.Handler

	GhostMaster *matrix.GhostMaster
	RoomManager *matrix.RoomManager
	Connector   BridgeConnector

	parentCtx       context.Context
	parentCtxCancel context.CancelFunc
}

// Implement Room callbacks
func (m *BridgeKit[T]) HandleMatrixEvent(room *matrix.Room, user bridge.User, event *event.Event) {
	m.handleMatrixRoomEvent(room, user, event)
}

func (m *BridgeKit[T]) HandleMarkEncrypted(room *matrix.Room) {
	if roomEventHandler, ok := m.Connector.(MatrixRoomEventHandler); ok {
		err := roomEventHandler.HandleMatrixMarkEncrypted(m.parentCtx, room)
		if err != nil {
			fmt.Println("Error handling MarkEncrypted event: ", err)
		}

		return
	}
}

func (m *BridgeKit[T]) UpdateBridgeInfo(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

// GetExampleConfig returns the example configuration for the BridgeKit.
func (m *BridgeKit[T]) GetExampleConfig() string {
	return m.exampleConfig
}

// GetConfigPtr returns a pointer to the configuration object associated with the BridgeKit.
func (m *BridgeKit[T]) GetConfigPtr() interface{} {
	fmt.Println("GetConfigPTR PTR")
	return m.Config.GetPtr(&m.Bridge.Config)
}

// Init initializes the BridgeKit, including the Connector, GhostMaster, RoomManager, and CommandProcessor.
func (m *BridgeKit[T]) Init() {
	fmt.Println("[Init]")
	if err := m.Connector.Init(m.parentCtx); err != nil {
		fmt.Println("Error initializing connector: ", err)
		return
	}

	m.GhostMaster = matrix.NewGhostMaster(&m.Bridge, m.localpart)
	m.RoomManager = matrix.NewRoomManager(&m.Bridge, m.GhostMaster, m)

	m.CommandProcessor = commands.NewProcessor(&m.Bridge)
	proc := m.CommandProcessor.(*commands.Processor)
	proc.AddHandlers(
		m.Commands...,
	)
}

// Start initializes the BridgeKit and starts the connector.
// It first waits for the websocket connection to be established,
// then starts the connector
func (m *BridgeKit[T]) Start() {
	fmt.Println("[Start]")

	m.WaitWebsocketConnected()
	m.Connector.Start(m.parentCtx)
}

// Stop stops the BridgeKit and its underlying Connector.
func (m *BridgeKit[T]) Stop() {
	fmt.Println("[Stop]")
	m.parentCtxCancel()
	m.Connector.Stop()
}

func (m *BridgeKit[T]) GetIPortal(roomID id.RoomID) bridge.Portal {
	room := m.Connector.GetRoom(m.parentCtx, roomID)
	if room != nil {
		m.RoomManager.LoadRoom(room)
	}
	return room
}

func (m *BridgeKit[T]) GetAllIPortals() []bridge.Portal {
	fmt.Println("[GetAllIPortals]")
	return m.Connector.GetAllRooms(m.parentCtx)
}

func (m *BridgeKit[T]) GetIUser(id id.UserID, create bool) bridge.User {
	fmt.Println("[GetIUser] ", id.String(), " create ", create)

	u := m.Connector.GetUser(m.parentCtx, id, create)
	if u != nil {
		u.BridgeState = m.NewBridgeStateQueue(u)
		u.SetManagementRoomHandler = m.SetManagementRoom
	}

	return u
}

func (m *BridgeKit[T]) IsGhost(userID id.UserID) bool {
	fmt.Println("[IsGhost] ", userID.String())
	return m.Connector.IsGhost(m.parentCtx, userID)
}

func (m *BridgeKit[T]) GetIGhost(userID id.UserID) bridge.Ghost {
	fmt.Println("[GetIGhost] ", userID.String())
	return m.Connector.GetGhost(m.parentCtx, userID)
}

func (m *BridgeKit[T]) CreatePrivatePortal(roomID id.RoomID, user bridge.User, ghost bridge.Ghost) {
	fmt.Println("[CreatePrivatePortal] -- roomID: ", roomID.String(), " user: ", user.GetMXID().String())
	//TODO implement me
	panic("implement me")
}

func (m *BridgeKit[T]) handleMatrixRoomEvent(room *matrix.Room, user bridge.User, evt *event.Event) {
	fmt.Println("[handleMatrixRoomEvent] ", room.Name, " evt: ", evt.Type)

	// check if connector implements RoomEventHandler with type assertion
	if roomEventHandler, ok := m.Connector.(MatrixRoomEventHandler); ok {
		err := roomEventHandler.HandleMatrixRoomEvent(m.parentCtx, room, user, evt)
		if err != nil {
			fmt.Println("Error handling room event: ", err)
		}

		return
	}

	fmt.Println("No room event handler")
}

// ReplyErrorMessage sends a notice message in the given room with the error message from the provided event.
// The message will be sent as a reply to the original event.
func (m *BridgeKit[T]) ReplyErrorMessage(ctx context.Context, evt *event.Event, room *matrix.Room, err error) (*mautrix.RespSendEvent, error) {

	content := &event.MessageEventContent{
		MsgType: event.MsgNotice,
		Body:    err.Error(),
	}
	content.SetReply(evt)
	return m.SendBotMessageInRoom(ctx, room, content)
}

// MarkRead marks the given event as read in the specified Matrix room using the provided intent.
// This allows marking messages as read as different users, such as ghosts or double puppets.
func (m *BridgeKit[T]) MarkRead(ctx context.Context, room *matrix.Room, eventID id.EventID, intent *appservice.IntentAPI) error {
	fmt.Println("Marking as read with intent: ", intent.UserID, " event: ", eventID.String())
	return intent.MarkRead(ctx, room.MXID, eventID)
}

// EditBotMessage edits a previously sent message by the bot in the specified room.
// It takes the room ID, the event ID of the message to edit, and the new content.
func (m *BridgeKit[T]) EditBotMessage(ctx context.Context, roomID id.RoomID, eventID id.EventID, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	// Set up the edit relation to the original message
	content.SetEdit(eventID)

	// Send the edit message using the bot's intent
	resp, err := m.Bot.SendMessageEvent(ctx, roomID, event.EventMessage, content)
	if err != nil {
		return nil, fmt.Errorf("failed to edit message: %w", err)
	}

	fmt.Println("Edited message: ", eventID.String(), " -> ", resp.EventID.String())
	return resp, nil
}

// MarkRead marks the given event as read in the specified Matrix room.
// It uses the bot's intent to mark the event as read, indicating that the bridge has read the event.
func (m *BridgeKit[T]) MarkBotRead(ctx context.Context, room *matrix.Room, evt *event.Event) error {
	fmt.Println("Marking as read: ", evt.ID.String())
	return m.Bot.MarkRead(ctx, room.MXID, evt.ID)
}

// UploadImage downloads an image from the provided URL and uploads it to Matrix.
// Returns the Matrix content URI of the uploaded image.
func (m *BridgeKit[T]) UploadImage(ctx context.Context, url string) (id.ContentURI, error) {
	bot := m.Bot

	getResp, err := http.DefaultClient.Get(url)
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to download image: %w", err)
	}

	data, err := io.ReadAll(getResp.Body)
	_ = getResp.Body.Close()
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to read image bytes: %w", err)
	}

	// Upload to Matrix
	resp, err := bot.UploadBytes(ctx, data, "image/png")
	if err != nil {
		return id.ContentURI{}, fmt.Errorf("failed to upload image to Matrix: %w", err)
	}
	fmt.Println("Uploaded image to Matrix servers -- ", resp.ContentURI.String())

	return resp.ContentURI, nil
}

// ResetRoomPermission resets the power levels for a given room, setting the bridge bot's power level to 9001 and the main intent user's power level to 100.
//
// ctx is the context to use for the operation.
// room is the Matrix room to reset the permissions for.
func (m *BridgeKit[T]) ResetRoomPermission(ctx context.Context, room *matrix.Room) (*mautrix.RespSendEvent, error) {
	fmt.Println("[ResetRoomPermission] ", room.Name)

	powerLevels := matrix.NewBasePowerLevels()
	powerLevels.Users = map[id.UserID]int{
		m.Bridge.Bot.UserID: 9001,
	}

	for _, ghost := range room.Ghosts {
		powerLevels.Users[ghost.MXID] = 100
	}

	resp, err := m.Bridge.Bot.SetPowerLevels(ctx, room.MXID, powerLevels)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// MarkRoomReadOnly sets the power levels in the given Matrix room to effectively make it read-only for the current user.
// This is done by setting the default power level to 101 and disabling reactions and messages for all users except the ghost and bot.
func (m *BridgeKit[T]) MarkRoomReadOnly(ctx context.Context, room *matrix.Room) (*mautrix.RespSendEvent, error) {
	fmt.Println("[MarkRoomReadOnly] ", room.Name)

	// set everyone to 100 except the current user, effectively takinga way his permission to do anything
	powerLevels := matrix.NewBasePowerLevels()
	powerLevels.Users = map[id.UserID]int{
		m.Bridge.Bot.UserID: 9001,
	}

	for _, ghost := range room.Ghosts {
		powerLevels.Users[ghost.MXID] = 102
	}

	// disable messages
	powerLevels.EventsDefault = 101
	powerLevels.Events[event.EventReaction.Type] = 101
	powerLevels.Events[event.EventMessage.Type] = 101

	resp, err := m.Bridge.Bot.SetPowerLevels(ctx, room.MXID, powerLevels)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateRoom creates a new Matrix room for the given portal and user. It invites the bot and the user to the room,
// sets the appropriate power levels, and updates the portal's MXID with the new room ID. It also updates the display names of any ghost users associated with the portal.
func (m *BridgeKit[T]) CreateRoom(ctx context.Context, portal *matrix.Room, user *matrix.User, avatarURL id.ContentURI) (*matrix.Room, *mautrix.RespCreateRoom, error) {
	userIdsToInvite := []id.UserID{
		m.Bot.UserID,
		user.MXID,
	}
	userIdsToInvite = append(userIdsToInvite, portal.GhostUserIDs()...)

	fmt.Println("Creating room with ids: ", userIdsToInvite)

	powerLevels := matrix.NewBasePowerLevels()
	powerLevels.Users = map[id.UserID]int{
		m.Bridge.Bot.UserID: 9001,
	}

	// set every ghost as normal power level
	for _, ghost := range portal.Ghosts {
		powerLevels.Users[ghost.GetMXID()] = 100
	}

	initialState := []*event.Event{{
		Type:    event.StatePowerLevels,
		Content: event.Content{Parsed: powerLevels},
	}}

	if m.Config.Bridge().GetEncryptionConfig().Default {
		fmt.Println("encryption is enabled, setting encryption event")
		evt := &event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1}
		if rot := m.Config.Bridge().GetEncryptionConfig().Rotation; rot.EnableCustom {
			evt.RotationPeriodMillis = rot.Milliseconds
			evt.RotationPeriodMessages = rot.Messages
		}

		initialState = append(initialState, &event.Event{
			Type: event.StateEncryption,
			Content: event.Content{
				Parsed: evt,
			},
		})
		portal.Encrypted = true
	}

	if !avatarURL.IsEmpty() {
		initialState = append(initialState, &event.Event{
			Type: event.StateRoomAvatar,
			Content: event.Content{
				Parsed: &event.RoomAvatarEventContent{URL: avatarURL.CUString()},
			},
		})
	}

	// if the user has a ordinary ghost available, we need to subtract it before doing the check
	numGhostsWithoutUser := len(portal.Ghosts)
	if m.GhostMaster.HasUserGhost(user) {
		// check if user ghost is part of the ghosts
		userGhost := m.GhostMaster.AsUserGhost(ctx, user)
		for _, ghost := range portal.Ghosts {
			if ghost.MXID != userGhost.UserID {
				numGhostsWithoutUser++
			}
		}
	}
	isPrivateChat := numGhostsWithoutUser == 1

	req := &mautrix.ReqCreateRoom{
		Visibility:            "private",
		Name:                  portal.Name,
		Topic:                 portal.Topic,
		Invite:                userIdsToInvite,
		Preset:                "private_chat",
		IsDirect:              isPrivateChat,
		BeeperAutoJoinInvites: true,
		PowerLevelOverride:    powerLevels,
		InitialState:          initialState,
	}

	room, err := m.Bridge.Bot.CreateRoom(ctx, req)
	if err != nil {
		fmt.Println("Error creating room: ", err)
		return nil, nil, err
	}

	fmt.Println("created room!! ", req.Name)
	portal.MXID = room.RoomID

	// also invite the user
	if err := m.RoomManager.AddUserToRoom(ctx, room.RoomID, user); err != nil {
		fmt.Println("Err adding user to room: ", err)
	}

	for _, ghost := range portal.Ghosts {
		if err := m.GhostMaster.UpdateGhostName(ctx, ghost, ghost.GetDisplayname()); err != nil {
			fmt.Println("Error updating ghost name: ", err)
		} else {
			fmt.Println("Updated ghost name: ", ghost.GetDisplayname())
		}
	}

	return portal, room, nil
}

// SendBotMessageInRoom sends a message in the given room on behalf of the bot.
// The content of the message is specified by the provided MessageEventContent.
// This is a convenience method that calls SendMessageInRoom with the bot as the sender.
func (m *BridgeKit[T]) SendBotMessageInRoom(ctx context.Context, room *matrix.Room, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	return m.SendMessageInRoom(ctx, room, m.Bot, content)
}

// SendTimestampedBotMessageInRoom sends a timestamped message from the bot to the given room.
func (m *BridgeKit[T]) SendTimestampedBotMessageInRoom(ctx context.Context, room *matrix.Room, content *event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	return m.SendTimestampedMessageInRoom(ctx, room, m.Bot, content, ts)
}

// BackfillMessages backfills a list of messages into the given Matrix room. If the Beeper feature for batch sending is supported, it will use that to send the messages in a single request. Otherwise, it will send each message individually.
//
// The `notify` parameter controls whether a notification should be sent for the backfilled messages.
// If `notify` is set to `true`, messages will not be marked as read
func (m *BridgeKit[T]) BackfillMessages(ctx context.Context, room *matrix.Room, user *matrix.User, msgs []*matrix.Message, notify bool) error {
	batchSending := m.SpecVersions.Supports(mautrix.BeeperFeatureBatchSending)

	// msgContent := format.RenderMarkdown(text, true, false)
	evs := []*event.Event{}
	for _, msg := range msgs {
		content := event.Content{
			Parsed: msg.Content,
		}

		if user.DoublePuppetIntent != nil {
			m.Bridge.Bot.AddDoublePuppetValue(&content)
		}

		evs = append(evs, &event.Event{
			Sender:    msg.FromMXID,
			Type:      event.EventMessage,
			Timestamp: msg.Timestamp,
			RoomID:    msg.RoomID,
			Content:   content,
			ToUserID:  msg.ToMXID,
		})
	}

	if batchSending {
		fmt.Println("Beeper batch sending enabled")
		req := &mautrix.ReqBeeperBatchSend{
			ForwardIfNoMessages: true,
			Forward:             true,
			SendNotification:    notify,
			Events:              evs,
		}
		if !notify {
			req.MarkReadBy = user.MXID
		}

		_, err := m.Bridge.Bot.BeeperBatchSend(ctx, room.MXID, req)
		if err != nil {
			fmt.Println("Error backfilling message: ", err)
			goto manualBackfill
			return err
		}

		return nil
	}

manualBackfill:
	for _, msg := range msgs {
		intent := m.GhostMaster.AsRoomGhostByID(room, msg.FromMXID)
		if msg.FromMXID == user.MXID {
			intent = m.GhostMaster.AsUserGhost(ctx, user)
		}

		_, err := m.SendTimestampedMainMessageInRoom(ctx, room, intent, msg.Content, msg.Timestamp)
		if err != nil {
			fmt.Println("err while inserting message: ", err.Error())
		}
	}

	return nil
}

// SendTimestampedMainMessageInRoom sends a message event with the given content and timestamp to the specified room, using the provided sender intent
func (m *BridgeKit[T]) SendTimestampedMainMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	if sender == nil {
		return nil, errors.New("no sender intent passed")
	}

	resp, err := sender.SendMassagedMessageEvent(ctx, room.MXID, event.EventMessage, content, ts)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendTimestampedUserMessageInRoom sends a message event with the given content and timestamp from the specified user in the given room.
func (m *BridgeKit[T]) SendTimestampedUserMessageInRoom(ctx context.Context, room *matrix.Room, user *matrix.User, content *event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	resp, err := m.GhostMaster.AsUserGhost(ctx, user).SendMassagedMessageEvent(ctx, room.MXID, event.EventMessage, content, ts)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendUserMessageInRoom sends a message event from the given user to the given room.
// The content of the message is specified by the provided MessageEventContent.
func (m *BridgeKit[T]) SendUserMessageInRoom(ctx context.Context, room *matrix.Room, user *matrix.User, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	resp, err := m.GhostMaster.AsUserGhost(ctx, user).SendMessageEvent(ctx, room.MXID, event.EventMessage, content)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendTimestampedMessageInRoom sends a message event with the given timestamp to the specified Matrix room, using the provided sender intent.
func (m *BridgeKit[T]) SendTimestampedMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content *event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	if sender == nil {
		return nil, errors.New("no sender intent passed")
	}

	resp, err := sender.SendMassagedMessageEvent(ctx, room.MXID, event.EventMessage, content, ts)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendMessageInRoom sends a message event to the given Matrix room using the provided sender.
func (m *BridgeKit[T]) SendMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	if sender == nil {
		return nil, errors.New("no sender intent passed")
	}

	resp, err := sender.SendMessageEvent(ctx, room.MXID, event.EventMessage, content)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// RegisterCommand adds a new chat command handler.
func (m *BridgeKit[T]) RegisterCommand(cmd commands.Handler) {
	m.Commands = append(m.Commands, cmd)
}

// StartBridgeConnector sets the given bridge connector and starts the event loop
func (m *BridgeKit[T]) StartBridgeConnector(ctx context.Context, connector BridgeConnector) {
	fmt.Println("[StartBridgeConnector] ")
	m.Connector = connector
	m.parentCtx, m.parentCtxCancel = context.WithCancel(ctx)
	m.Main()
}

func (m *BridgeKit[T]) SetManagementRoom(user *matrix.User, room id.RoomID) {
	fmt.Println("SetManagementRoom - ", user.DisplayName, room)
	m.Connector.SetManagementRoom(m.parentCtx, user, room)
}

func (m *BridgeKit[T]) encryptContent(ctx context.Context, room *matrix.Room, intent *appservice.IntentAPI, content *event.Content, eventType event.Type) (event.Type, error) {
	if !room.Encrypted || m.Bridge.Crypto == nil {
		return eventType, nil
	}
	intent.AddDoublePuppetValue(content)

	err := m.Bridge.Crypto.Encrypt(ctx, room.MXID, eventType, content)
	if err != nil {
		return eventType, fmt.Errorf("failed to encrypt event: %w", err)
	}
	return event.EventEncrypted, nil
}

func NewBridgeKit[T ConfigGetter](
	name, localpart, url, description, version string,
	conf T,
	exampleConfig string,
) *BridgeKit[T] {
	br := &BridgeKit[T]{
		localpart:     localpart,
		Config:        conf,
		exampleConfig: exampleConfig,
	}
	br.Bridge = bridge.Bridge{
		Name:        name,
		URL:         url,
		Description: description,
		Version:     version,

		CryptoPickleKey: "github.com/dvcrn/matrix-bridgekit",

		ConfigUpgrader: &configupgrade.StructUpgrader{
			SimpleUpgrader: configupgrade.SimpleUpgrader(conf.DoUpgrade),
			Base:           exampleConfig,
		},

		Child: br,
	}
	br.InitVersion(Tag, Commit, BuildTime)

	return br
}
