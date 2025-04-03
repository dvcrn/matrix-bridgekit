# Matrix BridgeKit - Amazon Q Guide

This document provides guidance for Amazon Q when working with this Matrix BridgeKit project.

## Project Overview

Matrix BridgeKit is an opinionated wrapper around mautrix-go that makes it easier to build Matrix bridges in Go. The project provides abstractions and pre-configurations to speed up bridge development.

## Key Components

- **BridgeConnector**: Interface that bridges should implement
- **MatrixRoomEventHandler**: Interface for handling Matrix room events
- **GhostMaster**: Manages ghost/puppet users
- **RoomManager**: Handles room creation and management

## Testing Instructions

To test if the connectors are working:

1. Build the boilerplate project:
   ```
   cd boilerplate && go build cmd/bridge/main.go
   ```

2. Run the boilerplate bridge:
   ```
   cd boilerplate && go run cmd/bridge/main.go
   ```

3. For a complete test with registration:
   ```
   cd boilerplate && go run cmd/bridge/main.go -r registration.yaml
   ```

## Common Issues

- Make sure the `HandleMatrixMarkEncrypted` method is implemented in any connector
- The `CreateRoom` method requires an avatar URL parameter
- Use `UpdateGhostName` instead of `UpdateName` for updating ghost names
- The `MarkRead` method signature requires room, eventID and intent parameters

## Glossary

- **Room/Portal**: A chat room (DM or group chat)
- **Puppet/Ghost**: A remote user
- **User**: A session/user using the bridge
- **Intent**: Representation of an entity (user, bot, ghost)

## Development Workflow

1. Implement the `bridgekit.BridgeConnector` and `bridgekit.MatrixRoomEventHandler` interfaces
2. Create a new BridgeKit instance with appropriate configuration
3. Initialize your connector with the BridgeKit instance
4. Start the bridge with `br.StartBridgeConnector(context.Background(), connector)`
