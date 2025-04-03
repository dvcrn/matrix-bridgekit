package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/bridge/commands"
	"maunium.net/go/mautrix/format"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/dvcrn/matrix-bridgekit/bridgekit"
	"github.com/dvcrn/matrix-bridgekit/matrix"
)

var _ bridgekit.BridgeConnector = (*MyBridgeConnector)(nil)
var _ bridgekit.MatrixRoomEventHandler = (*MyBridgeConnector)(nil)

type MemDB struct {
	Users map[id.UserID]*matrix.User
	Rooms map[id.RoomID]*matrix.Room
}

func (mdb *MemDB) Store() {
	data, err := json.Marshal(mdb)
	if err != nil {
		fmt.Println("Error marshaling database:", err)
		return
	}

	err = os.WriteFile("db.json", data, 0644)
	if err != nil {
		fmt.Println("Error writing database to file:", err)
		return
	}
}

func (mdb *MemDB) Load() {
	data, err := os.ReadFile("db.json")
	if err != nil {
		fmt.Println("Error reading database file:", err)
		return
	}

	err = json.Unmarshal(data, mdb)
	if err != nil {
		fmt.Println("Error unmarshaling database:", err)
		return
	}
}

func NewMemDB() *MemDB {
	return &MemDB{
		Users: map[id.UserID]*matrix.User{},
		Rooms: map[id.RoomID]*matrix.Room{},
	}
}

type MyBridgeConnector struct {
	kit   *bridgekit.BridgeKit[*Config]
	memDb *MemDB
}

func (m *MyBridgeConnector) Init(ctx context.Context) error {
	fmt.Println("Initializing MyBridgeConnector")
	m.memDb.Load()

	// No need to call LoadRoomIntent as it doesn't exist anymore

	m.kit.RegisterCommand(&commands.FullHandler{
		Func: func(e *commands.Event) {
			// dummy function function
			user := e.User.(*matrix.User)
			var room *matrix.Room
			if e.Portal != nil {
				room = e.Portal.(*matrix.Room)
			}
			fmt.Println("Login called", user.DisplayName, room.MXID.String())

			e.Reply("Okay, you logged in!!")

			// authenticate here. room is usually the management room
			// the user is already in the mem-DB because GetUser is called first
			// Let's create some rooms for them
			m.createDummyRooms(ctx, user)

		},
		Name: "login",
		Help: commands.HelpMeta{
			Section:     commands.HelpSectionAuth,
			Description: "Authenticate with the bridge",
		},
	})

	return nil
}

func (m *MyBridgeConnector) createDummyRooms(ctx context.Context, user *matrix.User) {
	ghost := m.kit.GhostMaster.NewGhost(
		"SomeUserID",
		"Test User",
		"user_name",
		id.ContentURI{},
	)

	room := m.kit.RoomManager.NewRoom("Test Room", "Some Topic", ghost)
	
	// Create an empty avatar URL for the fourth parameter
	emptyAvatarURL := id.ContentURI{}
	createdRoom, _, err := m.kit.CreateRoom(ctx, room, user, emptyAvatarURL)
	if err != nil {
		fmt.Println("err : ", err.Error())
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		// Use UpdateGhostName instead of UpdateName
		if err := m.kit.GhostMaster.UpdateGhostName(ctx, ghost, "Spooky Spooky Ghost"); err != nil {
			fmt.Println("err updating ghost name: ", err.Error())
		}

		content := format.RenderMarkdown("See? I can also update my own name", true, false)
		m.kit.SendMessageInRoom(ctx, createdRoom, createdRoom.MainIntent(), &content)
	}()

	content := format.RenderMarkdown("Hello, I'm a bot", true, false)
	m.kit.SendBotMessageInRoom(ctx, createdRoom, &content)

	content = format.RenderMarkdown("Hello, I'm a ghost", true, false)
	m.kit.SendMessageInRoom(ctx, createdRoom, createdRoom.MainIntent(), &content)
}

func (m *MyBridgeConnector) Start(ctx context.Context) {
	fmt.Println("Starting MyBridgeConnector")

	// Bind remote connection, eg websocket
	// Start interval polling
	// Create ghosts

	//bridgeConfig := m.kit.Config.Bridge.(MyBridgeConfig)
	//fmt.Printf("%v+\n", bridgeConfig.SomeKey)

	// TODO: for debug. remove me.
	func(v interface{}) {
		j, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		buf := bytes.NewBuffer(j)
		fmt.Printf("%v\n", buf.String())
	}(m.kit.Config)

	fmt.Printf("SomeKey: %v\n", m.kit.Config.BridgeConfig.SomeKey)
	fmt.Printf("SomeOtherSection Key%v\n", m.kit.Config.SomeOtherSection.Key)

	fmt.Println("--------------------")
	fmt.Println("Started bridge!! Go ahead and message", m.kit.Bot.UserID.String())
	fmt.Println("Use the login command :)")
	fmt.Println("--------------------")

}

func (m *MyBridgeConnector) Stop() {
	//TODO implement me
	fmt.Println("Stopping MyBridgeConnector")
	m.memDb.Store()
}

func (m *MyBridgeConnector) GetRoom(ctx context.Context, roomID id.RoomID) *matrix.Room {
	// check if in DB
	if r, ok := m.memDb.Rooms[roomID]; ok {
		return r
	}

	// not in db
	return &matrix.Room{
		MXID: roomID,
	}
}

func (m *MyBridgeConnector) GetAllRooms(ctx context.Context) []bridge.Portal {
	//TODO implement me
	panic("implement me")
}

func (m *MyBridgeConnector) IsGhost(ctx context.Context, userID id.UserID) bool {
	//TODO implement me
	return false
}

func (m *MyBridgeConnector) GetGhost(ctx context.Context, userID id.UserID) *matrix.Ghost {
	fmt.Println("GetGhost unimplemented")
	//TODO implement me
	return nil
}

func (m *MyBridgeConnector) GetUser(ctx context.Context, uid id.UserID, create bool) *matrix.User {
	if u, ok := m.memDb.Users[uid]; ok {
		return u
	}

	u := &matrix.User{
		MXID:             uid,
		RemoteID:         "whatsapp_id",
		RemoteName:       "WhatsApp Name",
		DisplayName:      "whatsapp_user",
		PermissionLevel:  bridgeconfig.PermissionLevelAdmin,
		ManagementRoomID: "", // dont have a management room id yet, otherwise return it here
	}

	m.memDb.Users[uid] = u

	return u
}

func (m *MyBridgeConnector) SetManagementRoom(ctx context.Context, user *matrix.User, roomID id.RoomID) {
	//TODO implement me
	fmt.Println("SetSetManagementRoom for ", user.DisplayName)
	if _, ok := m.memDb.Users[user.MXID]; ok {
		m.memDb.Users[user.MXID].ManagementRoomID = roomID
	}
}

func (m *MyBridgeConnector) HandleMatrixRoomEvent(ctx context.Context, room *matrix.Room, user bridge.User, evt *event.Event) error {
	switch evt.Type {
	case event.EventMessage:
		fmt.Println("got message event")
		// TODO: for debug. remove me.
		func(v interface{}) {
			j, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
			buf := bytes.NewBuffer(j)
			fmt.Printf("%v\n", buf.String())
		}(evt)
	default:
		fmt.Println("got unhandled event type: ", evt.Type.String())
	}

	// Update the MarkRead call to match the new signature
	if err := m.kit.MarkRead(ctx, room, evt.ID, m.kit.Bot); err != nil {
		fmt.Println("error marking as read: ", err)
	}
	return nil
}

// Add the missing HandleMatrixMarkEncrypted method
func (m *MyBridgeConnector) HandleMatrixMarkEncrypted(ctx context.Context, room *matrix.Room) error {
	fmt.Println("Room marked as encrypted:", room.MXID)
	// Implement any logic needed when a room is marked as encrypted
	return nil
}

func NewBridgeConnector(bk *bridgekit.BridgeKit[*Config]) *MyBridgeConnector {
	br := &MyBridgeConnector{
		kit:   bk,
		memDb: NewMemDB(),
	}
	return br
}
