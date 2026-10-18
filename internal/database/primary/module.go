package primary

import "go.uber.org/fx"

// Module wires the primary database into the application's Fx graph.
var Module = fx.Module("primary",
	fx.Provide(New),
)
