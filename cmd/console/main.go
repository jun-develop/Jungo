// Command console executes registered CLI commands.
//
// Usage:
//
//	make console CMD="user:list"
//	make console CMD="health:check"
//	make console CMD="user:list --verbose"   # also print app logs and executed SQL
//	make console CMD="user:list --debug"     # print every log, Fx events included
package main

import "jungo/internal/app"

func main() {
	app.NewConsoleFx().Run()
}
