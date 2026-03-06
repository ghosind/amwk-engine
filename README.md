# AMWK Framework — Engine

AMWK Engine provides the kernel component for the AMWK framework. It provides the request/response handling, middleware workflow, and more. The engine is a part of the framework, and it is designed to be adapter-agnostic, allowing it to work with various runtime environments.

It is for internal use by adapter implementations, please check the [Getting Started](#getting-started) section to see how to use it in your adapter.

## Requirements

- Recommended Go version: 1.22 or newer.

You can add the packages to your module-enabled project:

```bash
go get github.com/go-amwk/core
go get github.com/go-amwk/engine
```

## Getting Started

For adapter developers, you need to create your own adapter package base on your runtime environment, and provide your request and response implementations that implement the `core.Request` and `core.Response` interfaces.

It's a simple example to show how to use the `engine` package in your adapter's request handler:

```go
package main

import (
  "github.com/go-amwk/engine"
  "github.com/go-amwk/core"
)

// Example handler invoked by your runtime environment for each incoming request
func (app *App) HandleRequest(req *YourRequest, res *YourResponse) {
  // create request context for the incoming request and response
  ctx := engine.NewContext(app, req, res)

  // run the handlers
  if err := ctx.Next(); err != nil {
    // handle error
  }

  // send response via your runtime-specific response object
  // e.g. res.Send()
}
```

## Contributing

- Fork and clone the repository.
- Create a feature branch for your changes.
- Implement changes and add tests.
- Run tests and linters locally:
- Push your branch and open a pull request.

## License

This project is licensed under the Apache License 2.0. See the [`LICENSE`](./LICENSE) file for details.
