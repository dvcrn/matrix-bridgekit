package bridgekit

import (
	"context"
	"github.com/dvcrn/matrix-bridgekit/matrix"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type BridgeConnector interface {
	Init(ctx context.Context) error
	Start(ctx context.Context)
	Stop()

	GetRoom(ctx context.Context, roomID id.RoomID) *matrix.Room
	GetAllRooms(ctx context.Context) []bridge.Portal

	IsGhost(ctx context.Context, userID id.UserID) bool
	GetGhost(ctx context.Context, userID id.UserID) *matrix.Ghost
	GetUser(ctx context.Context, id id.UserID, create bool) *matrix.User

	SetManagementRoom(ctx context.Context, user *matrix.User, roomID id.RoomID)
}

type MatrixRoomEventHandler interface {
	HandleMatrixRoomEvent(ctx context.Context, room *matrix.Room, user bridge.User, evt *event.Event) error
}
