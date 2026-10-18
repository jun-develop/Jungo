// Command api runs the HTTP API server.
//
// Usage:
//
//	make app-dev
package main

import "jungo/internal/app"

func main() {
	app.NewAPIFx().Run()
}
