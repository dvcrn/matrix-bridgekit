package pkg

import (
	"context"

	"github.com/dvcrn/matrix-bridgekit/pkg/domain"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type BridgeConnector interface {
	Init(ctx context.Context) error
	Start(ctx context.Context)
	Stop()

	GetRoom(ctx context.Context, roomID id.RoomID) *domain.Room
	GetAllRooms(ctx context.Context) []bridge.Portal

	IsGhost(ctx context.Context, userID id.UserID) bool
	GetGhost(ctx context.Context, userID id.UserID) *domain.Ghost
	GetUser(ctx context.Context, id id.UserID, create bool) *domain.User

	SetManagementRoom(ctx context.Context, user *domain.User, roomID id.RoomID)
}

type MatrixRoomEventHandler interface {
	HandleMatrixRoomEvent(ctx context.Context, room *domain.Room, user bridge.User, evt *event.Event) error
}
