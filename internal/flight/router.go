package flight

import (
	"github.com/pocketbase/pocketbase/core"
)

// ConfigureRouting sets up the centralized routing for the Flight3 server
func ConfigureRouting(se *core.ServeEvent) {
	// SQLiter API routes
	se.Router.GET("/sqliter/file/*", HandleBanquetDownload)
	se.Router.GET("/sqliter/debug", HandleBanquetDebug)

	// Catch-all for Banquet
	se.Router.GET("/*", func(e *core.RequestEvent) error {
		// Default verbose to false
		return HandleBanquet(e, false)
	})
}
