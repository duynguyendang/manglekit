# Logging

MangleKit uses a structured logging system based on the `core.Logger` interface. This allows for consistent and configurable logging throughout the library.

## Standard Usage

The standard usage pattern is to use the logger provided by the `core.Observability` struct. This struct is passed to all components during initialization.

```go
package mypackage

import (
	"github.com/duynguyendang/manglekit/v1/core"
)

type MyComponent struct {
	logger core.Logger
}

func New(obs core.Observability) (*MyComponent, error) {
	return &MyComponent{
		logger: obs.Logger.With("component", "my-component"),
	}, nil
}

func (c *MyComponent) DoSomething() {
	c.logger.Infof("doing something")
}
```

## Logging Levels

The `core.Logger` interface supports the following logging levels:

- `Debugf`: Used for verbose, fine-grained logging.
- `Infof`: Used for informational messages.
- `Warnf`: Used for warnings that don't prevent the system from functioning.
- `Errorf`: Used for errors that prevent the system from functioning.

## Injecting Custom Loggers

You can inject a custom logger by implementing the `core.Logger` interface and passing it to the `Builder`.

```go
package main

import (
	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
)

type MyLogger struct{}

func (l *MyLogger) Debugf(msg string, kv ...any) {
	// ...
}

func (l *MyLogger) Infof(msg string, kv ...any) {
	// ...
}

func (l *MyLogger) Warnf(msg string, kv ...any) {
	// ...
}

func (l *MyLogger) Errorf(msg string, kv ...any) {
	// ...
}

func (l *MyLogger) With(kv ...any) core.Logger {
	// ...
}

func main() {
	builder := manglekit.NewBuilder()
	builder.WithObservability(core.Observability{
		Logger: &MyLogger{},
	})
	// ...
}
```
