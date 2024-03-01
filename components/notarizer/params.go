package notarizer

import (
	"github.com/iotaledger/hive.go/app"
)

// ParametersRestAPI contains the definition of the parameters used by the inx-notarizer HTTP server.
type ParametersRestAPI struct {
	// BindAddress defines the bind address on which the inx-notarizer HTTP server listens.
	BindAddress string `default:"localhost:9687" usage:"the bind address on which the inx-notarizer HTTP server listens"`

	// AdvertiseAddress defines the address of the inx-notarizer HTTP server which is advertised to the INX Server (optional).
	AdvertiseAddress string `default:"" usage:"the address of the inx-notarizer HTTP server which is advertised to the INX Server (optional)"`

	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled.
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`

	// MetadataCacheSize defines the size of the cache for each IRC standard.
	MetadataCacheSize int `default:"1000" usage:"defines the size of the cache for each IRC standard"`
}

var ParamsRestAPI = &ParametersRestAPI{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"restAPI": ParamsRestAPI,
	},
	Masked: nil,
}
