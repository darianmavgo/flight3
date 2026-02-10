package flight

import (
	"github.com/pocketbase/pocketbase/core"
)

// ConfigureRouting sets up the HTTP routes for the application.
// It centralizes routing configuration.
func ConfigureRouting(se *core.ServeEvent) {
	// Client endpoints
	se.Router.GET("/sqliter/home", HandleHome)
	se.Router.GET("/sqliter/file/*", HandleBanquetDownload)
	se.Router.GET("/sqliter/debug", HandleBanquetDebug)

	// Default fallback to Banquet handler
	// This captures everything else and tries to treat it as a Banquet URL
	se.Router.GET("/*", func(e *core.RequestEvent) error {
		// Pass verbose=false by default
		return HandleBanquet(e, false)
	})
}
