package matrix

import (
	"context"
	"fmt"

	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var _ bridge.Portal = &Room{}

type MatrixEventHandler func(room *Room, user bridge.User, event *event.Event)

type Room struct {
	RemotedID string    `json:"remoted_id,omitempty"`
	MXID      id.RoomID `json:"mxid,omitempty"`
	Name      string    `json:"name,omitempty"`
	Topic     string    `json:"topic,omitempty"`

	Encrypted   bool `json:"encrypted,omitempty"`
	PrivateChat bool `json:"private_chat,omitempty"`

	BotIntent *appservice.IntentAPI `json:"-"`
	Ghosts    []*Ghost              `json:"ghosts,omitempty"`

	MatrixEventHandler MatrixEventHandler `json:"-"`
}

func NewRoom(name string, topic string, botIntent *appservice.IntentAPI, ghosts ...*Ghost) *Room {
	return &Room{
		Encrypted:   false,
		PrivateChat: len(ghosts) == 1,
		//MXID:        roomID,
		Name:  name,
		Topic: topic,

		Ghosts:    ghosts,
		BotIntent: botIntent,
	}
}

func (p *Room) GhostUserIDs() []id.UserID {
	ghostIDs := make([]id.UserID, len(p.Ghosts))
	for i, ghost := range p.Ghosts {
		ghostIDs[i] = ghost.MXID
	}

	return ghostIDs
}

// IsEncrypted implements bridge.Portal.
func (p *Room) IsEncrypted() bool {
	return p.Encrypted
}

// IsPrivateChat implements bridge.Portal.
func (p *Room) IsPrivateChat() bool {
	fmt.Println("[IsPrivateChat] ", p.Name, p.PrivateChat)
	return p.PrivateChat
}

// MainIntent returns the reference to who should be sending the message, a ghost or the bot.
// If the Room is a private chatroom, then the message should come from the other user
// otherwise it should come from the bot
func (p *Room) MainIntent() *appservice.IntentAPI {
	if p.IsPrivateChat() {
		return p.Ghosts[0].DefaultIntent()
	}

	return p.BotIntent
}

// MarkEncrypted implements bridge.Portal.
func (p *Room) MarkEncrypted() {
	panic("MarkEncrypted unimplemented")
}

// ReceiveMatrixEvent implements bridge.Portal.
func (p *Room) ReceiveMatrixEvent(user bridge.User, evt *event.Event) {
	if p.MatrixEventHandler != nil {
		p.MatrixEventHandler(p, user, evt)
	}
}

// UpdateBridgeInfo implements bridge.Portal.
func (p *Room) UpdateBridgeInfo(ctx context.Context) {
	panic("UpdateBridgeInfo unimplemented")
}
