# zerrors - Typed Domain Errors for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/DeluxeOwl/zerrors.svg)](https://pkg.go.dev/github.com/DeluxeOwl/zerrors)

`zerrors` provides a flexible and type-safe way to define and work with domain-specific errors in Go applications. It integrates seamlessly with Go's standard error handling (`errors.Is`, `errors.As`, `errors.Unwrap`) and provides structured logging out-of-the-box via `slog`.

## Why zerrors?

- **Typed Error Codes:** Define domain-specific error codes using custom string types, enhancing code clarity and enabling compile-time checks.
- **Structured Context:** Attach arbitrary key-value data (`With`) and searchable tags (`Tags`) to errors for better debugging and observability.
- **Standard Error Compatibility:** Works flawlessly with `errors.Is`, `errors.As`, and `errors.Unwrap`, allowing gradual adoption and interoperability with other error libraries.
- **Rich Logging:** Implements `slog.LogValuer` to automatically log errors with code, message, structured data, tags, wrapped error details, and stack traces.
- **Error Wrapping:** Easily wrap underlying errors while preserving context and type information.
- **Stack Traces:** Automatically captures concise stack traces upon error creation (excluding noisy runtime/test frames).
- **Exhaustive Error Handling:** Facilitates mapping specific error codes to distinct actions (e.g., different HTTP status codes or user messages in a handler). Using `errors.As` combined with a `switch` on the `.Code()` allows you to handle each domain error case explicitly. Linters like [`exhaustive`](https://golangci-lint.run/usage/linters/#exhaustive) can then ensure you've handled all defined error codes, preventing bugs when new codes are introduced.

## Features

- Generic error type `Error[T ~string]` for typed error codes.
- Chainable methods for adding context: `WithError`, `Errorf`, `With`, `Tags`.
- Full `errors.Is`, `errors.As`, `errors.Unwrap` support.
- `slog.LogValuer` implementation for structured logging.
- Helper functions `As` (type-safe casting with callback) and `HasCode` (check code existence in chain).
- Automatic and clean stack trace capture.

## Installation

```bash
go get github.com/DeluxeOwl/zerrors
```

## Usage

### Defining Domain Errors

Define your domain-specific error codes using a custom string type.

```go
package services

import "github.com/DeluxeOwl/zerrors" // Use your actual path

type UserServiceError string

const (
    ErrUserNotFound      UserServiceError = "user_not_found"
    ErrInvalidUserData   UserServiceError = "invalid_user_data"
    ErrPermissionDenied  UserServiceError = "permission_denied"
)

type DBError string

const (
    ErrDBConnection DBError = "db_connection_failed"
    ErrDBNotFound   DBError = "db_record_not_found"
)
```

### Creating and Enriching Errors

Create errors using `zerrors.New` and add context.

```go
import (
    "fmt"
    "log/slog"
    "os"

    "github.com/DeluxeOwl/zerrors"
    "your_project/services"
    "your_project/storage"
)


func findUser(userID int) error {
    // Simulate a database error
    dbErr := zerrors.New(storage.ErrDBNotFound).
        With("query", "SELECT * FROM users WHERE id = ?").
        With("attempt", 1).
        Tags("database", "read-replica")

    // Wrap the database error with a service-level error
    userServiceErr := zerrors.New(services.ErrUserNotFound).
        With("user_id", userID).
        With("trace_id", "abc-123").
        Tags("critical", "lookup").
        WithError(dbErr) // Wrap the original error

    return userServiceErr
}

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    err := findUser(12345)

    if err != nil {
        // Log the error using slog - zerrors.Error automatically provides structured data
        logger.Error("Failed to find user", slog.Any("error", err))

        // Output (format might vary slightly):
        // {
        //   "time": "...",
        //   "level": "ERROR",
        //   "msg": "Failed to find user",
        //   "error": {
        //     "code": "user_not_found",
        //     "error": "user_not_found: db_record_not_found: SELECT * FROM users WHERE id = ?",
        //     "data": {
        //       "trace_id": "abc-123",
        //       "user_id": 12345
        //     },
        //     "tags": [ "critical", "lookup", "database", "read-replica" ], // Order might vary
        //     "wrapped": {
        //       "code": "db_record_not_found",
        //       "error": "db_record_not_found: SELECT * FROM users WHERE id = ?",
        //       "data": {
        //         "attempt": 1,
        //         "query": "SELECT * FROM users WHERE id = ?"
        //       },
        //       "tags": [ "database", "read-replica" ] // Order might vary
        //     },
        //     "stack": "\n    at your_project/main.findUser(main.go:25)\n    at main.main(main.go:35)\n    ..."
        //   }
        // }
    }
}
```

### Working with `errors.Is` and `errors.As`

`zerrors` integrates seamlessly with the standard Go error handling functions.

```go
import (
    "errors"
    "fmt"

    "github.com/DeluxeOwl/zerrors"
    "your_project/services"
    "your_project/storage"
)

func processError(err error) {
    // 1. Check for specific error codes using HasCode (Recommended for checking codes in the chain)
    if zerrors.HasCode(err, services.ErrUserNotFound) {
        fmt.Println("Handling: User not found scenario")
        // Extract specific error details if needed
        var userErr *zerrors.Error[services.UserServiceError]
        if errors.As(err, &userErr) {
             if userID, ok := userErr.Get("user_id"); ok {
                 fmt.Printf("  Details: User ID %v\n", userID)
             }
        }
    }

    if zerrors.HasCode(err, storage.ErrDBNotFound) {
         fmt.Println("Handling: Underlying DB record not found")
    }

    // 2. Using errors.As to get the specific typed error
    var serviceErr *zerrors.Error[services.UserServiceError]
    if errors.As(err, &serviceErr) {
        fmt.Printf("Service Error Code: %s\n", serviceErr.Code())
        if traceID, ok := serviceErr.Get("trace_id"); ok {
            fmt.Printf("  Trace ID: %s\n", traceID)
        }
        if serviceErr.HasTags("critical") {
             fmt.Println("  Tagged as critical!")
        }
    }

    var dbErr *zerrors.Error[storage.DBError]
    if errors.As(err, &dbErr) { // errors.As unwraps automatically
        fmt.Printf("DB Error Code: %s\n", dbErr.Code())
        if query, ok := dbErr.Get("query"); ok {
            fmt.Printf("  DB Query: %s\n", query)
        }
    }

    // 3. Using errors.Is (Checks if the error *is* a specific instance)
    // Note: The custom Is method compares codes *only if* the target is also a *zerrors.Error*
    // of the *same* underlying type T. For general code checking in a chain, use HasCode or errors.As.
    sentinelErr := zerrors.New(services.ErrPermissionDenied)
    if errors.Is(err, sentinelErr) {
         fmt.Println("Error is specifically ErrPermissionDenied (at the top level or matching code)")
    }

    // 4. Using the type-safe `As` helper
    if _, found := zerrors.As(err, func(e *zerrors.Error[services.UserServiceError]) any {
        fmt.Printf("Found UserServiceError via helper: Code=%s\n", e.Code())
        // Perform actions directly with the typed error 'e'
        return nil // Return value isn't used here, just demonstrating the callback
    }); found {
         fmt.Println("  Successfully processed via As helper.")
    }

    // 5. Unwrapping
    underlying := errors.Unwrap(err)
    if underlying != nil {
        fmt.Printf("Underlying error: %s\n", underlying.Error()) // Prints the wrapped error's message
        // You can then check the underlying error further
        var dbErrCheck *zerrors.Error[storage.DBError]
        if errors.As(underlying, &dbErrCheck) {
             fmt.Printf("  Confirmed underlying is DBError with code: %s\n", dbErrCheck.Code())
        }
    }
}

func main() {
    err := findUser(12345) // Function from previous example
    if err != nil {
        processError(err)
    }
}
```

### Tagging

Tags provide a simple way to categorize errors. Tags are automatically propagated when wrapping errors.

```go
errDb := zerrors.New(storage.ErrDBConnection).Tags("database", "transient")
errSvc := zerrors.New(services.ErrPermissionDenied).
    Tags("security", "authz").
    WithError(errDb)

fmt.Println("Service Tags:", errSvc.GetTags()) // Output: [security authz database transient] (order may vary)
fmt.Println("Has 'database' tag:", errSvc.HasTags("database")) // Output: true
fmt.Println("Has 'security' AND 'transient':", errSvc.HasTags("security", "transient")) // Output: true
fmt.Println("Has 'unknown' tag:", errSvc.HasTags("unknown")) // Output: false
```

## Key Concepts Summary

- **Typed Codes (`T ~string`)**: Use custom types for error codes (e.g., `type MyErrorCode string`) for better domain modeling.
- **Standard Compatibility**: Leverages `errors.Is`, `errors.As`, `errors.Unwrap` for maximum interoperability. Use `HasCode` or `errors.As` for robust checking within error chains.
- **Structured Logging (`slog`)**: Errors log rich context automatically when passed to `slog` (e.g., `slog.Error("operation failed", slog.Any("error", err))`).
- **Stack Traces**: Captured on `New`, included in `slog` output, and accessible via `stack.String()` (though typically only used during logging).

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.
