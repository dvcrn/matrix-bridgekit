# BridgeKit to bridgev2 Migration Design Document

## 1. Introduction

This document outlines the comprehensive plan to migrate the existing BridgeKit framework to utilize mautrix/go's bridgev2 architecture. The migration aims to leverage the improved architecture, performance, and features of bridgev2 while maintaining compatibility with existing bridges built on BridgeKit.

## 2. Current Architecture (BridgeKit)

BridgeKit is currently an opinionated wrapper around mautrix/go that provides:

- A `BridgeConnector` interface for bridges to implement
- Domain-style object design (vs. active-record style)
- Separation of side-effect operations into dedicated managers
- Pre-configured patterns for common bridge operations

The current architecture uses:
- A central `BridgeKit` struct inheriting from `bridge.Bridge`
- Helper classes like `GhostMaster` and `RoomManager`
- A connector-based pattern for integration with different platforms

Key limitations include:
- Less clear separation between Matrix and network concerns
- Tighter coupling between components
- Less flexible identity management

## 3. Target Architecture (bridgev2)

bridgev2 uses a modular, component-based architecture with three primary subsystems:

1. **Matrix Connector** - Handles all Matrix protocol interactions
2. **Network Connector** - Manages connections to the bridged platform
3. **Central Bridge Module** - Coordinates between the two connectors

The new architecture features:
- Clearly defined interfaces for different components
- Robust identity management via the `networkid` package
- Improved state management and persistence
- Support for advanced features like disappearing messages
- Better background processing capabilities

## 4. Core Components to Migrate

### 4.1 Bridge Controller

**Current:** 
- `BridgeKit` struct with embedded `bridge.Bridge`
- Manages connectors, ghosts, and rooms directly

**Target:**
- `Bridge` struct in bridgev2
- Clearer separation of responsibilities
- More systematic object lifecycle management

**Changes Needed:**
- Create a new `BridgeKitV2` struct that uses the bridgev2 `Bridge`
- Move core functionality from BridgeKit to appropriate components in new structure
- Adapt initialization flow to match bridgev2 patterns

### 4.2 Connector Interfaces

**Current:**
- `BridgeConnector` interface with mixed responsibilities
- `MatrixRoomEventHandler` for Matrix events

**Target:**
- `MatrixConnector` for Matrix-side operations
- `NetworkConnector` for remote-network operations
- Clear separation between Matrix and network responsibilities

**Changes Needed:**
- Define adapter interfaces to translate between BridgeConnector and NetworkConnector
- Implement a default MatrixConnector for standard Matrix operations
- Split event handling between Matrix and network sides

### 4.3 Entity Models

**Current:**
- `Room`/`Portal` for chat rooms
- `User` for Matrix users
- `Ghost`/`Puppet` for remote users

**Target:**
- `Portal` with enhanced capabilities
- `User` with login management
- `UserLogin` to represent network connections
- `Ghost` with improved identity handling

**Changes Needed:**
- Update entity models to match bridgev2 patterns
- Implement data migration for existing installations
- Create mapper functions between old and new models

### 4.4 Database Layer

**Current:**
- Direct database access patterns
- Simple state persistence

**Target:**
- Structured database package
- Migration system
- Key-value storage for bridge state

**Changes Needed:**
- Adapt to new database patterns
- Implement migration scripts for existing data
- Update persistence logic throughout codebase

### 4.5 Event Flow

**Current:**
- Direct event handling in connector
- Simple command processing

**Target:**
- More sophisticated event routing
- Better context handling
- Improved error management

**Changes Needed:**
- Redefine event handling patterns
- Implement new command processor
- Update context propagation throughout the codebase

## 5. Interface Adaptations

### 5.1 Bridge Connector Interface

**Current:**
```go
type BridgeConnector interface {
    Init(context.Context) error
    Start(context.Context)
    Stop()
    
    GetRoom(context.Context, id.RoomID) *matrix.Room
    GetAllRooms(context.Context) []bridge.Portal
    GetUser(context.Context, id.UserID, bool) bridge.User
    IsGhost(context.Context, id.UserID) bool
    GetGhost(context.Context, id.UserID) bridge.Ghost
    SetManagementRoom(context.Context, *matrix.User, id.RoomID)
}
```

**Target:**
```go
type NetworkConnector interface {
    Init(*Bridge)
    Start(context.Context) error
    GetDBMetaTypes() database.MetaTypes
    GetCapabilities() NetworkCapabilities
    GetBridgeInfoVersion() (int, int)
    
    // Network-specific operations
    CreateUserClient(context.Context, *User) (NetworkAPI, error)
    // Additional methods for network-specific operations
}
```

**Migration Strategy:**
- Create adapter layer to translate between interfaces
- Implement NetworkConnector for existing bridges
- Provide backward compatibility utilities

### 5.2 Event Handler Interface

**Current:**
```go
type MatrixRoomEventHandler interface {
    HandleMatrixRoomEvent(context.Context, *matrix.Room, bridge.User, *event.Event) error
    HandleMatrixMarkEncrypted(context.Context, *matrix.Room) error
}
```

**Target:**
Will be split between MatrixConnector event handling and NetworkConnector event handling.

**Migration Strategy:**
- Create adapter pattern for event routing
- Map existing handlers to appropriate interfaces
- Update documentation for implementers

### 5.3 User and Ghost Interfaces

**Current:**
Direct use of matrix.User and matrix.Ghost

**Target:**
Structured User, UserLogin, and Ghost types with clear interfaces

**Migration Strategy:**
- Create wrapper types for backward compatibility
- Update core interfaces to use new types
- Provide utilities for conversion

## 6. Configuration Changes

### 6.1 Bridge Configuration

**Current:**
- Bridge uses `ConfigGetter` interface
- Configuration handling through `GetConfigPtr`

**Target:**
- Uses `bridgeconfig.BridgeConfig`
- More structured configuration approach

**Changes Needed:**
- Adapt configuration handling to match bridgev2 patterns
- Implement converters for existing configurations
- Update documentation for config changes

### 6.2 Feature Configuration

**Current:**
- Limited feature flags
- Manual capability handling

**Target:**
- More structured capability system
- Better feature toggling

**Changes Needed:**
- Implement capability mapping
- Update feature detection logic
- Provide defaults for new capabilities

## 7. Utility Functions Migration

### 7.1 Ghost Management

**Current:**
- `GhostMaster` handles ghost creation and management

**Target:**
- Ghost management through MatrixConnector
- More systematic identity handling

**Changes Needed:**
- Adapt GhostMaster functionality to new patterns
- Update ghost creation and management logic
- Implement conversion utilities

### 7.2 Room Management

**Current:**
- `RoomManager` handles room operations

**Target:**
- Portal management through Bridge and connectors
- More systematic room handling

**Changes Needed:**
- Adapt RoomManager functionality to bridge methods
- Update room creation and management logic
- Implement conversion utilities

### 7.3 Message Handling

**Current:**
- Direct message sending through matrix API

**Target:**
- More structured message flow
- Better error handling

**Changes Needed:**
- Update message sending patterns
- Implement new error handling
- Adapt to new context patterns

## 8. Implementation Plan

### 8.1 Phase 1: Core Architecture

1. Create new package structure matching bridgev2
2. Implement core Bridge structure
3. Define adapter interfaces
4. Set up basic connector patterns

### 8.2 Phase 2: Data Model and Database

1. Update entity models to match bridgev2
2. Implement database migration utilities
3. Create data mapping functions
4. Test data conversion

### 8.3 Phase 3: Event Flow and Commands

1. Implement new event handling patterns
2. Create command processor
3. Update context propagation
4. Test event routing

### 8.4 Phase 4: Compatibility Layer

1. Create backward compatibility utilities
2. Implement adapter patterns for existing bridges
3. Update documentation
4. Create migration guides

### 8.5 Phase 5: Testing and Validation

1. Test with sample bridges
2. Validate data migration
3. Performance testing
4. Security review

## 9. Migration Challenges

### 9.1 Breaking Changes

- Interface changes will require updates to implementing bridges
- Data model changes may require migration scripts
- Event flow changes will affect existing logic

### 9.2 Compatibility Considerations

- Provide transition period with both architectures
- Implement compatibility layers where possible
- Create clear migration guides

### 9.3 Performance Impacts

- Analyze performance changes from architecture shift
- Identify potential bottlenecks
- Optimize critical paths

## 10. Testing Strategy

### 10.1 Unit Testing

- Test each component in isolation
- Validate interface implementations
- Ensure data integrity

### 10.2 Integration Testing

- Test component interactions
- Validate event flow
- Ensure proper error propagation

### 10.3 Migration Testing

- Test data migration scripts
- Validate backward compatibility
- Ensure smooth transition

## 11. Documentation Updates

### 11.1 API Documentation

- Update all interface documentation
- Document new patterns and best practices
- Provide migration examples

### 11.2 Migration Guides

- Create step-by-step guides for existing bridges
- Document common patterns and solutions
- Provide troubleshooting information

### 11.3 Example Implementations

- Update example bridges to use new architecture
- Provide reference implementations
- Document key design decisions

## 12. Conclusion

Migrating from BridgeKit to bridgev2 represents a significant architectural upgrade that will improve maintainability, performance, and feature support. The modular design of bridgev2 enables clearer separation of concerns, better testability, and more flexible integration patterns.

By following this migration plan, we can ensure a smooth transition while maintaining compatibility with existing implementations where possible. The resulting framework will provide a more robust foundation for building Matrix bridges in the future.
