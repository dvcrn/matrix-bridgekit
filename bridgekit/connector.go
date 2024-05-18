package bridgekit

import (
	"context"

	"github.com/dvcrn/matrix-bridgekit/matrix"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type BridgeConnector interface {
	// Init initialises the connector. Use this to set everything up
	Init(ctx context.Context) error
	// Start starts the default bridge functionality. Use this to set up loops, sockets, etc
	Start(ctx context.Context)
	// Stop shuts the bridge down
	Stop()

	// GetRoom returns the matrix room with the given ID.
	GetRoom(ctx context.Context, roomID id.RoomID) *matrix.Room
	// GetAllRooms returns all rooms that the bridge has access to.
	GetAllRooms(ctx context.Context) []bridge.Portal

	// IsGhost returns whether the given userid is a ghost
	IsGhost(ctx context.Context, userID id.UserID) bool
	// GetGhost returns the ghost with the given ID
	GetGhost(ctx context.Context, userID id.UserID) *matrix.Ghost
	// GetUser returns the user with the given ID. If create is true, it should create a new user if it doesn't exist.
	GetUser(ctx context.Context, id id.UserID, create bool) *matrix.User

	// SetManagementRoom sets the management room for the given user.
	SetManagementRoom(ctx context.Context, user *matrix.User, roomID id.RoomID)
}

type MatrixRoomEventHandler interface {
	// HandleMatrixRoomEvent is the callback to handle matrix events within a specific room.
	HandleMatrixRoomEvent(ctx context.Context, room *matrix.Room, user bridge.User, evt *event.Event) error
}
