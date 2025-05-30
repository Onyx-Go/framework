# Onyx

A Laravel-inspired web framework for Go, designed to bring the elegance and developer experience of Laravel to the Go ecosystem.

## Features

- **Expressive Routing**: Laravel-style route definitions with parameter binding
- **Middleware Stack**: Powerful middleware system for request/response processing
- **Dependency Injection**: Built-in container for managing dependencies
- **Database ORM**: Eloquent-inspired ORM for database operations
- **Configuration Management**: Environment-based configuration system
- **Template Engine**: Built-in templating with layout support
- **CLI Tooling**: Artisan-inspired commands for scaffolding and development
- **Validation**: Request validation with custom rules
- **Logging**: Structured logging with multiple channels

## Quick Start

```go
package main

import "github.com/onyx-go/framework"

func main() {
    app := framework.New()
    
    app.Get("/", func(c *framework.Context) error {
        return c.String(200, "Hello Onyx!")
    })
    
    app.Start(":8080")
}
```

## Directory Structure

```
app/
├── Controllers/     # HTTP controllers
├── Models/         # Database models
├── Middleware/     # Custom middleware
└── Services/       # Business logic services

config/             # Configuration files
database/
├── migrations/     # Database migrations
└── seeds/         # Database seeders

routes/             # Route definitions
├── web.go         # Web routes
└── api.go         # API routes

resources/
├── views/         # Template files
└── assets/        # Static assets

storage/
├── logs/          # Log files
└── cache/         # Cache files

tests/             # Test files
```