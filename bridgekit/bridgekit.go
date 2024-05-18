package bridgekit

import (
	"context"
	_ "embed"
	"fmt"

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
	DoUpgrade(*configupgrade.Helper)
	GetPtr(*bridgeconfig.BaseConfig) any
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
	m.RoomManager = matrix.NewRoomManager(&m.Bridge, m.GhostMaster)

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
		m.RoomManager.LoadRoomIntent(room)
		m.bindRoomHandlers(room)
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

// bindRoomHandlers binds the matrix room event handler for the given room.
// this function is needed to attach room handlers to room objects that are freshly created
func (m *BridgeKit[T]) bindRoomHandlers(room *matrix.Room) {
	fmt.Println("[bindRoomHandlers] ", room.Name)
	room.MatrixEventHandler = m.handleMatrixRoomEvent
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

// MarkRead marks the given event as read in the specified Matrix room.
// It uses the bot's intent to mark the event as read, indicating that the bridge has read the event.
func (m *BridgeKit[T]) MarkRead(ctx context.Context, evt *event.Event, room *matrix.Room) error {
	fmt.Println("Marking as read: ", evt.ID.String())

	return room.BotIntent.MarkRead(ctx, room.MXID, evt.ID)
}

// ResetRoomPermission resets the power levels for a given room, setting the bridge bot's power level to 9001 and the main intent user's power level to 100.
//
// ctx is the context to use for the operation.
// room is the Matrix room to reset the permissions for.
func (m *BridgeKit[T]) ResetRoomPermission(ctx context.Context, room *matrix.Room) (*mautrix.RespSendEvent, error) {
	fmt.Println("[ResetRoomPermission] ", room.Name)

	powerLevels := matrix.NewBasePowerLevels()
	powerLevels.Users = map[id.UserID]int{
		room.MainIntent().UserID: 100,
		m.Bridge.Bot.UserID:      9001,
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
		room.MainIntent().UserID: 102,
		m.Bridge.Bot.UserID:      9001,
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
func (m *BridgeKit[T]) CreateRoom(ctx context.Context, portal *matrix.Room, user *matrix.User) (*matrix.Room, *mautrix.RespCreateRoom, error) {
	userIdsToInvite := []id.UserID{
		m.Bot.UserID,
		user.MXID,
	}
	userIdsToInvite = append(userIdsToInvite, portal.GhostUserIDs()...)

	fmt.Println("Creating room with ids: ", userIdsToInvite)

	powerLevels := matrix.NewBasePowerLevels()
	powerLevels.Users = map[id.UserID]int{
		portal.MainIntent().UserID: 100,
		m.Bridge.Bot.UserID:        9001,
	}

	req := &mautrix.ReqCreateRoom{
		Visibility:            "private",
		Name:                  portal.Name,
		Topic:                 portal.Topic,
		Invite:                userIdsToInvite,
		Preset:                "private_chat",
		IsDirect:              portal.IsPrivateChat(),
		BeeperAutoJoinInvites: true,
		PowerLevelOverride:    powerLevels,
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
		if err := m.GhostMaster.UpdateName(ctx, ghost, ghost.GetDisplayname()); err != nil {
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
		evs = append(evs, &event.Event{
			Sender:    msg.FromMXID,
			Type:      event.EventMessage,
			Timestamp: msg.Timestamp,
			RoomID:    msg.RoomID,
			Content: event.Content{
				Parsed: msg.Content,
			},
			ToUserID: msg.ToMXID,
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

		_, err := room.BotIntent.BeeperBatchSend(ctx, room.MXID, req)
		if err != nil {
			fmt.Println("Error backfilling message: ", err)
			goto manualBackfill
			return err
		}

		return nil
	}

manualBackfill:
	for _, msg := range msgs {
		intent := room.MainIntent()
		if msg.FromMXID == user.MXID {
			intent = user.DoublePuppetIntent
		}

		_, err := m.SendTimestampedMainMessageInRoom(ctx, room, intent, msg.Content, msg.Timestamp)
		if err != nil {
			fmt.Println("err while inserting message: ", err.Error())
		}
	}

	return nil
}

// SendTimestampedMainMessageInRoom sends a message event with the given content and timestamp to the specified room, using the provided sender intent if not nil, or the room's main intent if sender is nil.
func (m *BridgeKit[T]) SendTimestampedMainMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	intent := sender
	if intent == nil {
		intent = room.MainIntent()
	}

	resp, err := intent.SendMassagedMessageEvent(ctx, room.MXID, event.EventMessage, content, ts)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendTimestampedUserMessageInRoom sends a message event with the given content and timestamp from the specified user in the given room.
func (m *BridgeKit[T]) SendTimestampedUserMessageInRoom(ctx context.Context, room *matrix.Room, user *matrix.User, content *event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	resp, err := m.AS.Client(user.MXID).SendMessageEvent(ctx, room.MXID, event.EventMessage, content, mautrix.ReqSendEvent{
		Timestamp: ts,
	})
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendUserMessageInRoom sends a message event from the given user to the given room.
// The content of the message is specified by the provided MessageEventContent.
func (m *BridgeKit[T]) SendUserMessageInRoom(ctx context.Context, room *matrix.Room, user *matrix.User, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {

	resp, err := m.AS.Client(user.MXID).SendMessageEvent(ctx, room.MXID, event.EventMessage, content)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendTimestampedMessageInRoom sends a message event with the given timestamp to the specified Matrix room, using the provided sender intent.
// If the sender intent is nil, the room's main intent will be used instead.
func (m *BridgeKit[T]) SendTimestampedMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content *event.MessageEventContent, ts int64) (*mautrix.RespSendEvent, error) {
	intent := sender
	if intent == nil {
		intent = room.MainIntent()
	}

	resp, err := intent.SendMassagedMessageEvent(ctx, room.MXID, event.EventMessage, content, ts)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return nil, err
	}

	return resp, nil
}

// SendMessageInRoom sends a message event to the given Matrix room using the provided sender.
// If no sender is provided, the room's main intent is used.
func (m *BridgeKit[T]) SendMessageInRoom(ctx context.Context, room *matrix.Room, sender *appservice.IntentAPI, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	intent := sender
	if intent == nil {
		intent = room.MainIntent()
	}

	resp, err := intent.SendMessageEvent(ctx, room.MXID, event.EventMessage, content)
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
