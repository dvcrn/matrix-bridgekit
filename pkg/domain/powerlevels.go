package domain

import (
	"maunium.net/go/mautrix/event"
)

func NewBasePowerLevels() *event.PowerLevelsEventContent {
	var anyone = 0
	var disabled = 99

	return &event.PowerLevelsEventContent{
		UsersDefault:    anyone,
		EventsDefault:   anyone,
		RedactPtr:       &anyone,
		StateDefaultPtr: &disabled,
		BanPtr:          &disabled,
		KickPtr:         &disabled,
		InvitePtr:       &disabled,
		Events: map[string]int{
			event.StateRoomName.Type:   anyone,
			event.StateRoomAvatar.Type: anyone,
			event.EventReaction.Type:   anyone,
			event.EventRedaction.Type:  anyone,
			event.EventMessage.Type:    anyone,
		},
	}
}
