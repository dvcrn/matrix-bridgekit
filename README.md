## Matrix BridgeKit

Build matrix bridges quickly.

## What is this?

BridgeKit is a _opinionated_ wrapper around [mautrix-go](https://github.com/mautrix/go) with the goal of making it easier to build matrix bridges.

All the heavy lifting is done by mautrix/go, this library just tries to make it easier to use by providing opinionated abstractions, with the option to always fall back to mautrix-go.

## Notes

- Bridge can only create things within it's namespace, so for example in my test ist was `sh-localtest_` as the namespace for all userids
- In a 1-to-1 chat, a portal will reflect a chatroom with 1 user, and a puppet in it 


## Glossary 

- **Portal**: A "chat room", doesn't matter what it is. Can be a DM or a group chat
- **Puppet**/**Ghost**: A remote user
- **User**: Basically a session, aka a user that is using the bridge 