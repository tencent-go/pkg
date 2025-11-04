## Description
This package provides a type-safe configuration management system that automatically parses environment variables into Go structs.

## Usage
```go
type Config struct {
    DatabaseURL string `env:"DB_URL" default:"localhost:5432" example:"postgres://user:pass@localhost:5432/db" description:"Database connection URL"`
    MaxRetries  int    `env:"MAX_RETRIES" default:"3" example:"5" description:"Maximum number of retry attempts"`
    Debug       bool   `env:"DEBUG" default:"false" description:"Enable debug mode"`
}
```

## Tags

### env
Specifies the environment variable key name.

- Use `-` to ignore the field
- Add `,omitempty` to make the field optional

### default
Default value for the field. If specified, the field becomes optional.

### example
Example value for documentation purposes.

### description
Field description for documentation purposes.

## Error Handling
When configuration parsing fails, the system will:
1. Generate a detailed error report showing all configuration fields
2. Display current values, defaults, and any parsing errors
3. Panic with a "Configuration parsing error" message

## Supported Types
- String
- Integer (int, int8, int16, int32, int64)
- Unsigned Integer (uint, uint8, uint16, uint32, uint64)
- Float (float32, float64)
- Boolean
- Slice (comma-separated values)
- Array (fixed-size arrays, comma-separated values)
- Pointer types
- Nested structs

Note: For array and slice types, values should be provided as comma-separated strings in the environment variable. For example:
```go
type Config struct {
    Ports []int `env:"PORTS" default:"8080,8081,8082" description:"List of ports to listen on"`
    IPs   [2]string `env:"IPS" default:"127.0.0.1,0.0.0.0" description:"Fixed array of IP addresses"`
}
```

## Interface Implementation Check

To check if a type implements an interface at compile time, you can use the following pattern:

```go
// Method 1: Using var declaration
var _ InterfaceName = (*YourType)(nil)

// Method 2: Using type assertion
func checkInterface[T any]() {
    var _ T = (*YourType)(nil)
}
```

Example:
```go
type Config interface {
    Get() any
}

// This will cause a compile error if MyConfig doesn't implement Config
var _ Config = (*MyConfig)(nil)
```

The compiler will report an error if the type doesn't implement all methods of the interface. This is a common Go idiom for compile-time interface implementation checking.

Note: This check is performed at compile time, not runtime. It's a zero-cost way to ensure interface compliance.
