# BridgeKit to bridgev2 Migration Design Document

## 1. Introduction

This document outlines the plan to migrate the existing BridgeKit framework to utilize mautrix/go's bridgev2 architecture. The migration will be implemented in phases to ensure a smooth transition while maintaining backward compatibility where possible.

The bridgev2 architecture offers several significant improvements:
- Clear separation between Matrix and network concerns
- More robust identity management
- Improved state management and persistence
- Support for advanced features like disappearing messages and threads
- Better background processing capabilities

## 2. Current Architecture (BridgeKit)

BridgeKit is currently an opinionated wrapper around mautrix/go that provides:

- A `BridgeConnector` interface for bridges to implement
- A central `BridgeKit` struct inheriting from `bridge.Bridge`
- Helper classes like `GhostMaster` and `RoomManager`
- A connector-based pattern for integration with different platforms

The current implementation has limitations:
- Less clear separation between Matrix and network concerns
- Tighter coupling between components
- Less flexible identity management
- Limited support for advanced features

## 3. Target Architecture (bridgev2)

bridgev2 uses a modular, component-based architecture with three primary subsystems:

1. **Matrix Connector** - Handles all Matrix protocol interactions
2. **Network Connector** - Manages connections to the bridged platform
3. **Central Bridge Module** - Coordinates between the two connectors

The new architecture provides:
- Clearly defined interfaces for different components
- Robust identity management via the `networkid` package
- Improved state management and persistence
- Support for advanced features like disappearing messages
- Better background processing capabilities

## 4. Migration Phases

### Phase 1: Foundation Setup and Core Interfaces

**Goal:** Create the basic scaffolding and core interfaces for BridgeKitV2

#### Core Bridge Structure
- Create a new `BridgeKitV2` struct based on bridgev2.Bridge
- Design an initialization flow compatible with bridgev2
- Set up configuration handling that supports both old and new config formats
- Implement bridge lifecycle management (start, stop, etc.)

#### Interface Definitions
- Create an adapter that implements NetworkConnector but delegates to BridgeConnector
- Define a default MatrixConnector implementation that handles Matrix operations
- Establish a clear mapping between old concepts and new ones
- Document the interface changes for bridge developers

#### Database Preparation
- Set up the bridgev2 database structure
- Create schema migration helpers for existing data
- Establish data conversion utilities between old and new models
- Implement version tracking for upgrades

#### Technical Approach:
```go
// Core bridge structure in bridgekitv2/bridge.go
type BridgeKitV2[T ConfigGetter] struct {
    br *bridgev2.Bridge
    Config T
    
    // Adapter components
    matrixConnector MatrixConnector
    networkAdapter  NetworkConnectorAdapter
    
    // Migration utilities
    migrationHelpers *MigrationHelpers
}

// Adapter to convert BridgeConnector to NetworkConnector
type NetworkConnectorAdapter struct {
    connector BridgeConnector
    bridge    *BridgeKitV2
    // Additional state needed for bridgev2 compatibility
}

func (a *NetworkConnectorAdapter) Init(br *bridgev2.Bridge) {
    // Initialize the adapter
    // Store reference to the bridge
}

// Implement other NetworkConnector methods...
```

### Phase 2: Entity Model Migration

**Goal:** Update the object models to be compatible with bridgev2

#### Portal/Room Transition
- Adapt the current Room model to use the `Portal` structure from bridgev2
- Implement required Portal methods for bridgev2 compatibility
- Create mapping and conversion functions between old Room and new Portal
- Support portal key generation and lookup

#### User Model Updates
- Migrate the User structure to match bridgev2 requirements
- Implement UserLogin functionality to represent network connections
- Update user authentication and management flows
- Support the bridge state mechanism

#### Ghost Model Transformation
- Update Ghost handling to use bridgev2's identity management
- Implement proper identity mapping with the networkid package
- Create conversion utilities for existing ghost data
- Add support for additional ghost metadata

#### Technical Approach:
```go
// Room to Portal adapter
type PortalAdapter struct {
    *bridgev2.Portal
    room *matrix.Room  // Reference to legacy room
}

// User adapter
type UserAdapter struct {
    *bridgev2.User
    legacyUser *matrix.User
}

// Ghost adapter
type GhostAdapter struct {
    *bridgev2.Ghost
    legacyGhost *matrix.Ghost
}

// Portal key generation
func GeneratePortalKey(roomID string, remoteID string) networkid.PortalKey {
    return networkid.NewPortalKey(
        networkid.NewNetworkID("bridgekit"),
        networkid.ChatID(remoteID),
    )
}
```

### Phase 3: Event and Message Handling

**Goal:** Implement the new event processing flow

#### Basic Message Conversion
- Implement ConvertedMessage/ConvertedMessagePart functionality
- Create handlers for message creation, editing, and deletion
- Support different message types (text, media, etc.)
- Implement proper metadata handling

#### Event Routing
- Update event handling to use bridgev2 event flow
- Implement backfill mechanisms for history
- Add support for reactions and other extended events
- Create proper context management for event processing

#### Command Processing
- Update command processing to match bridgev2 patterns
- Create a compatible CommandProcessor implementation
- Migrate existing commands to the new system
- Support for advanced command features

#### Technical Approach:
```go
// Message conversion
func ConvertMatrixMessage(ctx context.Context, evt *event.Event) *bridgev2.ConvertedMessage {
    // Convert Matrix event to bridgev2 message format
    parts := []*bridgev2.ConvertedMessagePart{}
    
    // Handle different event types
    switch evt.Type {
    case event.EventMessage:
        // Convert message event
        content := evt.Content.AsMessage()
        parts = append(parts, &bridgev2.ConvertedMessagePart{
            Type:    evt.Type,
            Content: content,
            // Additional fields...
        })
    // Handle other event types...
    }
    
    return &bridgev2.ConvertedMessage{
        Parts: parts,
        // Additional fields...
    }
}

// Event routing adapter
func (a *NetworkConnectorAdapter) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
    // Convert bridgev2 message to format expected by BridgeConnector
    // Call appropriate methods on the underlying connector
    // Convert response back to bridgev2 format
}
```

### Phase 4: Advanced Features and Polishing

**Goal:** Implement the more advanced bridgev2 features and finalize the API

#### Additional Capabilities
- Implement disappearing messages support
- Add threads support
- Support for edits, reactions, and other advanced features
- Implement additional optional interfaces as needed

#### Compatibility Layer
- Finalize backward compatibility helpers
- Create migration guides for bridge developers
- Provide sample code for different migration scenarios
- Implement graceful fallbacks for unsupported features

#### Documentation and Testing
- Update the boilerplate example to use bridgev2
- Create comprehensive documentation
- Add test cases for key functionality
- Document migration paths

#### Technical Approach:
```go
// Disappearing message support
func (a *NetworkConnectorAdapter) GetCapabilities() bridgev2.NetworkGeneralCapabilities {
    return bridgev2.NetworkGeneralCapabilities{
        DisappearingMessages: true,
        // Other capabilities...
    }
}

// Thread support
func (a *NetworkConnectorAdapter) HandleMatrixThreadedReply(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
    // Handle threaded replies
    // Delegate to appropriate connector methods
}

// Migration guide example
// Example of how to migrate an existing bridge:
/*
// Before:
type MyBridge struct {
    kit *bridgekit.BridgeKit[*Config]
}

// After:
type MyBridge struct {
    kit *bridgekitv2.BridgeKitV2[*Config]
}

// Migration steps:
1. Update imports to use bridgekitv2
2. Review and update connector implementations
3. Run database migration utilities
*/
```

### Phase 5: Performance Optimization and Deployment

**Goal:** Finalize the implementation and prepare for production

#### Performance Testing
- Identify and resolve bottlenecks
- Optimize database queries
- Reduce memory consumption
- Benchmark key operations

#### Final API Review
- Lock down the public API surface
- Ensure all interfaces are properly documented
- Handle edge cases and error conditions
- Validate compatibility with existing bridges

#### Sample Migration
- Create a complete migration example with the boilerplate
- Provide upgrade scripts for existing installations
- Document breaking changes and workarounds
- Create a comprehensive test suite

## 5. Compatibility Considerations

The migration will aim to maintain compatibility with existing bridges in the following ways:

1. **Interface Adaptation**:
   - Provide adapter layers that translate between old and new interfaces
   - Allow gradual migration of code bases

2. **Database Migration**:
   - Create automated migration tools for existing data
   - Support both old and new schemas during transition

3. **API Compatibility**:
   - Maintain method signatures where possible
   - Provide compatibility wrappers for changed APIs

4. **Documentation**:
   - Clearly document migration paths
   - Provide examples of before/after code

## 6. Timeline and Milestones

- **Phase 1 (Foundation)**: 2-3 weeks
- **Phase 2 (Entity Models)**: 2-3 weeks
- **Phase 3 (Event Handling)**: 2-3 weeks
- **Phase 4 (Advanced Features)**: 2-3 weeks
- **Phase 5 (Optimization)**: 1-2 weeks

Total estimated time: 9-14 weeks

## 7. Risks and Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing bridges | Provide comprehensive migration guides and compatibility layers |
| Data loss during migration | Create robust database migration utilities with validation |
| Performance degradation | Benchmark critical operations and optimize as needed |
| Complex implementation | Maintain clear architecture documentation and code examples |
| Feature parity gaps | Identify missing features early and prioritize implementation |

## 8. Conclusion

Migrating from BridgeKit to bridgev2 represents a significant architectural upgrade that will improve maintainability, performance, and feature support. The modular design of bridgev2 enables clearer separation of concerns, better testability, and more flexible integration patterns.

By following this phased migration plan, we can ensure a smooth transition while maintaining compatibility with existing implementations where possible. The resulting framework will provide a more robust foundation for building Matrix bridges in the future.
