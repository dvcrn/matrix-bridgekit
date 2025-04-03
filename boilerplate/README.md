## matrix bridgekit boilerplate

Boilerplate template for [github.com/dvcrn/matrix-bridgekit](https://github.com/dvcrn/matrix-bridgekit)

## What is this?

BridgeKit is a _opinionated_ light wrapper around [mautrix-go](https://github.com/mautrix/go).

All the heavy lifting is done by mautrix/go, this library just tries to make it quicker to use by providing opinionated abstractions, with the option to always fall back to mautrix-go.

Super WIP

## Getting started

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

## Notes

- Bridge can only create things within it's namespace, so for example if your bridge is `sh-mybridge`, all ghosts have to be under `sh-mybridge_xxxxx`

## Glossary

- **Room**/**Portal**: A "chat room", doesn't matter what it is. Can be a DM or a group chat, everything is a room
- **Puppet**/**Ghost**: A remote user
- **User**: Basically a session, aka a user that is using the bridge
- **Intent**: Representation of a entity (user, bot, ghost). eg "BotIntent" is the handle to do something as the Bot
