## Matrix BridgeKit

Build matrix bridges in Go quickly

## What is this?

BridgeKit is a _opinionated_ wrapper around [mautrix-go](https://github.com/mautrix/go).

All the heavy lifting is done by mautrix/go, this library just tries to make it quicker to use by providing opinionated abstractions, with the option to always fall back to mautrix-go.

Super WIP

**What this wrapper does**:

- Provide a `Connector` interface that bridges should implement, rather than extending the bridge interface
- Decouple responsibilities of structs
  - Switch room/puppet/user from active-record style to domain style (no logic in domain objects)
  - Move methods that call matrix servers into bridge/roommanager/ghostmaster to make it more clear what has side effects
- Opinionated pre-configuration to make getting started a tad faster

## Getting started

[Check out the boilerplate](github.com/dvcrn/bridgekit-boilerplate) on what you have to do. The gist is, implement the `bridgekit.BridgeConnector` and `bridgekit.MatrixRoomEventHandler` interfaces, then load the connector with bridgekit:

```golang
var _ bridgekit.BridgeConnector = (*MyBridgeConnector)(nil)
var _ bridgekit.MatrixRoomEventHandler = (*MyBridgeConnector)(nil)

type MyBridgeConnector struct {
	kit   *bridgekit.BridgeKit[*Config]
}

func NewBridgeConnector(bk *bridgekit.BridgeKit[*Config]) *MyBridgeConnector {
	br := &MyBridgeConnector{
		kit:   bk,
	}
	return br
}

// ... implement all the required methods by the interface

func main() {
	br := bridgekit.NewBridgeKit(
		"MyBridge",
		"sh-mybridge",
		"",
		"Some cool integration",
		"1.0",
		&internal.Config{},
		internal.ExampleConfig,
	)
	connector := internal.NewBridgeConnector(br)
	br.StartBridgeConnector(context.Background(), connector)
}
```

### Connect the bridge to beeper

1. Make sure you have bbctl (https://github.com/beeper/bridge-manager) setup
2. Generate a registration file: `bbctl register -o  registration.yaml sh-mycoolbridge`
3. Copy example-config.yml to config.yml: `cp internal/example-config.yaml config.yaml`
4. Fill out the fields in config.yaml with information from registration.yaml:
   - Set `id`, `url`, `as_token`, `hs_token`, `bot.username`, `address`, `appservice.address`
   - `bot.username` has to match your bridge namespace. If your bridge is `sh-mycoolbridge`, the bot will be `sh-mycoolbridgebot`
   - `address` should match your personal beeper homeserver address that the registration commandg gave you, eg `https://user.eu-plucky-sparrow.edge.beeper.com/davicorn`
   - domain should be `beeper.local`
5. Update registration.yaml: Change `url` to the localhost value that your bridge is configured to

Run the bridge: `go run cmd/bridge/main.go -r registration.yaml`

If everything went fine, your bridge should connect and you you should be able to message the bridgebot to get started

## How to basics

## Notes

- Bridge can only create things within it's namespace, so for example if your bridge is `sh-mybridge`, all ghosts have to be under `sh-mybridge_xxxxx`

## Glossary

- **Room**/**Portal**: A "chat room", doesn't matter what it is. Can be a DM or a group chat, everything is a room
- **Puppet**/**Ghost**: A remote user
- **User**: Basically a session, aka a user that is using the bridge
- **Intent**: Representation of a entity (user, bot, ghost). eg "BotIntent" is the handle to do something as the Bot
