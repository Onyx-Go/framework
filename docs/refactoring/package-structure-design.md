# Onyx Framework Package Structure Design

## Current State Analysis
- **64 Go files** in root package (35,828+ lines)
- All functionality mixed in single `github.com/onyx-go/framework` package
- Large monolithic files violating Go best practices
- Poor separation of concerns

## Target Package Structure (Go Best Practices)

```
github.com/onyx-go/framework/
├── cmd/
│   └── onyx/                    # CLI application
│       ├── main.go
│       └── commands/
│           ├── make/            # Code generation commands
│           ├── migrate/         # Database migration commands
│           ├── serve/           # Development server
│           └── docs/            # Documentation commands
├── pkg/
│   └── onyx/                    # Public API (current pkg/onyx/)
│       ├── onyx.go              # Main framework interface
│       ├── types.go             # Public types
│       └── interfaces.go        # Public interfaces
├── internal/
│   ├── app/                     # Application bootstrapping
│   │   ├── application.go       # From application.go
│   │   ├── container.go         # From container.go
│   │   └── lifecycle.go         # Startup/shutdown logic
│   ├── http/                    # HTTP layer
│   │   ├── router/              # From router.go
│   │   ├── context/             # From context.go
│   │   ├── middleware/          # Middleware implementations
│   │   └── response/            # Response handling
│   ├── database/                # Database layer
│   │   ├── connection/          # Connection management
│   │   ├── orm/                 # ORM functionality
│   │   │   ├── query/           # Query builder
│   │   │   ├── model/           # Model operations
│   │   │   └── relations/       # Relationship handling
│   │   ├── migration/           # From migrations.go
│   │   └── events/              # From model_events.go
│   ├── config/                  # From config.go
│   │   ├── loader.go            # Configuration loading
│   │   ├── validator.go         # Configuration validation
│   │   └── cache.go             # Configuration caching
│   ├── security/                # Security features
│   │   ├── auth/                # From auth.go, authorization.go
│   │   ├── csrf/                # From csrf.go
│   │   ├── middleware/          # From security_middleware.go
│   │   └── validation/          # Security validation
│   ├── logging/                 # From logging.go
│   │   ├── logger.go            # Core logging
│   │   ├── channels/            # Log channels
│   │   └── formatters/          # Log formatters
│   ├── cache/                   # From cache.go, response_cache.go
│   │   ├── store/               # Cache storage
│   │   ├── serializer/          # Cache serialization
│   │   └── middleware/          # Cache middleware
│   ├── queue/                   # From queue.go
│   │   ├── driver/              # Queue drivers
│   │   ├── job/                 # Job definitions
│   │   └── worker/              # Job workers
│   ├── scheduler/               # From scheduler.go
│   │   ├── cron/                # Cron scheduling
│   │   ├── job/                 # Scheduled jobs
│   │   └── manager/             # Schedule manager
│   ├── mail/                    # From mail.go
│   │   ├── driver/              # Mail drivers
│   │   ├── template/            # Mail templates
│   │   └── queue/               # Queued mail
│   ├── storage/                 # From storage.go
│   │   ├── disk/                # Disk storage
│   │   ├── cloud/               # Cloud storage
│   │   └── cache/               # Storage caching
│   ├── template/                # From template.go
│   │   ├── engine/              # Template engine
│   │   ├── compiler/            # Template compilation
│   │   └── cache/               # Template caching
│   ├── validation/              # From validation.go
│   │   ├── rules/               # Validation rules
│   │   ├── messages/            # Validation messages
│   │   └── sanitizer/           # Input sanitization
│   ├── docs/                    # Documentation generation
│   │   ├── openapi/             # From openapi.go, api_docs.go
│   │   ├── swagger/             # From swagger_*.go
│   │   ├── annotations/         # From api_annotations.go
│   │   └── versioning/          # From api_versioning.go
│   └── testing/                 # From testing.go
│       ├── http/                # HTTP testing helpers
│       ├── database/            # Database testing helpers
│       └── mock/                # Mock implementations
└── testdata/                    # Test fixtures and data
    ├── migrations/              # Test migrations
    ├── seeds/                   # Test seeds
    └── fixtures/                # Test fixtures
```

## Package Responsibilities

### `cmd/onyx/` - CLI Application
- **Single Responsibility**: Command-line interface
- **Dependencies**: Internal packages only
- **Go Convention**: All CLI apps go in `cmd/`

### `pkg/onyx/` - Public API
- **Single Responsibility**: Framework's public interface
- **Dependencies**: Internal packages via interfaces
- **Go Convention**: Public APIs go in `pkg/`

### `internal/app/` - Application Core
- **Files**: `application.go`, `container.go`
- **Responsibility**: App lifecycle, DI container
- **Interfaces**: `Container`, `Application`

### `internal/http/` - HTTP Handling
- **Files**: `router.go`, `context.go`, middleware files
- **Responsibility**: HTTP request/response handling
- **Interfaces**: `Router`, `Context`, `Middleware`, `Handler`

### `internal/database/` - Data Layer
- **Files**: `database.go`, `migrations.go`, `relationships.go`, etc.
- **Responsibility**: Database operations, ORM, migrations
- **Interfaces**: `Database`, `QueryBuilder`, `Model`, `Migrator`

### `internal/config/` - Configuration
- **Files**: `config.go`, `database_config.go`
- **Responsibility**: Configuration management
- **Interfaces**: `ConfigLoader`, `ConfigValidator`

### `internal/security/` - Security
- **Files**: `security.go`, `auth.go`, `csrf.go`, etc.
- **Responsibility**: Authentication, authorization, security
- **Interfaces**: `Authenticator`, `Authorizer`, `CSRFProtector`

## Migration Strategy

### Phase 1: Create Package Structure
1. Create all `internal/` directories
2. Move files to appropriate packages
3. Update import statements
4. Fix circular dependencies

### Phase 2: Extract Interfaces
1. Define clean interfaces for each package
2. Implement dependency injection
3. Remove circular dependencies
4. Add proper error handling

### Phase 3: Testing
1. Ensure all tests pass after each move
2. Add missing tests for new packages
3. Verify no functionality regression

## Benefits of New Structure

### Go Best Practices Compliance
- ✅ Clear separation of concerns
- ✅ Single responsibility per package
- ✅ Proper use of `internal/` vs `pkg/`
- ✅ Domain-driven package organization

### Maintainability Improvements
- ✅ Smaller, focused files
- ✅ Clear dependency graph
- ✅ Easier to understand and modify
- ✅ Better testability

### Performance Benefits
- ✅ Reduced compilation times
- ✅ Better caching of compiled packages
- ✅ Easier to optimize specific components

## Implementation Notes

### Import Path Changes
- Current: `github.com/onyx-go/framework`
- New: `github.com/onyx-go/framework/internal/...`
- Public: `github.com/onyx-go/framework/pkg/onyx`

### Interface Definitions
Each package will define clean interfaces to prevent circular dependencies and improve testability.

### Backward Compatibility
The `pkg/onyx/` package will re-export all public APIs to maintain backward compatibility during transition.