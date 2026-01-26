package controller

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// MainHandler serves as the central routing logic for Flight3.
// Rules:
// 1. /_/ routes go straight to PocketBase (untouched).
// 2. No path ("/") translates to Banquet root handling.
// 3. All other requests go to handleBanquet (unless specific API routes match first).
func MainHandler(evt *core.RequestEvent, fallbackHandler func(*core.RequestEvent) error) error {
	path := evt.Request.URL.Path

	// Rule 1: /_/ goes straight to PocketBase
	// PocketBase registers /_/* routes itself, so usually they are matched before a wildcard /* handler.
	// However, if we are inside a wildcard handler "/*", we must confirm we aren't hijacking it.
	// But since we are likely defining this logic in the "/*" route or a middleware:
	if strings.HasPrefix(path, "/_/") {
		// Just continue to next handler (which should be PB's internal handler if using middleware)
		// Or if this is a final handler, we return nil to let PB default router handle it?
		// Actually, if we are in MainHandler called by a route, we can't "pass back" easily unless we use Next().
		return evt.Next()
	}

	// Rule 2: Root "/" -> handleBanquet
	// Normalized root handling.
	if path == "/" || path == "" {
		// We want to serve the local root explicitly.
		// We modify the request to look like a Banquet request for local root.
		// Assuming user "local" and host "localhost".
		// This is a specific logic from main.go we are moving here.
		targetURI := "/https:/local@localhost/"

		// Mutate request for the handler
		evt.Request.RequestURI = targetURI
		evt.Request.URL.Path = targetURI
		return fallbackHandler(evt)
	}

	// Rule 3: All other requests -> handleBanquet
	// This assumes specific API routes (/api/rclone/...) are registered BEFORE this handler
	// and thus won't reach here.
	return fallbackHandler(evt)
}

// BindRoutes registers the controller logic to the app router.
// banquetHandler is the closure that calls handleBanquet with dependencies.
func BindRoutes(e *core.ServeEvent, banquetHandler func(*core.RequestEvent) error) {
	// Register explicit API routes first (if any are separate)
	// ... logic for /api/rclone/browse is separate usually.

	// Explicit Root
	e.Router.GET("/", func(evt *core.RequestEvent) error {
		return MainHandler(evt, banquetHandler)
	})

	// Catch-All
	e.Router.GET("/*", func(evt *core.RequestEvent) error {
		// Special Check: If the path is /_/..., do we even reach here?
		// PocketBase routes are usually registered on the same mux.
		// If PB registers /_/, Go 1.22 matches most specific. so /_/ matches /_/.
		// But /_/... matches /_/{path...} which PB likely registers.
		// So this catch-all shouldn't catch /_/.
		// But MainHandler has checks just in case.
		return MainHandler(evt, banquetHandler)
	})
}
