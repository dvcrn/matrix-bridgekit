package matrix

import (
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Message struct {
	FromMXID  id.UserID                 `json:"from_mxid,omitempty"`
	ToMXID    id.UserID                 `json:"to_mxid,omitempty"`
	RoomID    id.RoomID                 `json:"room_id,omitempty"`
	Content   event.MessageEventContent `json:"content"`
	Timestamp int64                     `json:"timestamp,omitempty"`
}
