package notarizer

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"go.uber.org/dig"

	"github.com/adrian-grassl/inx-notarizer/pkg/daemon"
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/joho/godotenv"
)

func init() {
	Component = &app.Component{
		Name:     "Notarizer",
		Params:   params,
		DepsFunc: func(cDeps dependencies) { deps = cDeps },
		Run:      run,
	}
}

var (
	Component *app.Component
	deps      dependencies
)

type dependencies struct {
	dig.In
	NodeBridge *nodebridge.NodeBridge
}

func run() error {
	// create a background worker that handles the API
	if err := Component.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		Component.LogInfo("Starting API ... done")

		e := httpserver.NewEcho(Component.Logger(), nil, ParamsRestAPI.DebugRequestLoggerEnabled)

		Component.LogInfo("Starting API server ...")

		setupRoutes(e)

		LoadEnvVariables()

		go func() {
			Component.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := e.Start(ParamsRestAPI.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				Component.LogErrorfAndExit("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := ParamsRestAPI.BindAddress
		if ParamsRestAPI.AdvertiseAddress != "" {
			advertisedAddress = ParamsRestAPI.AdvertiseAddress
		}

		routeName := strings.Replace(APIRoute, "/api/", "", 1)

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, routeName, advertisedAddress, APIRoute); err != nil {
			Component.LogErrorfAndExit("Registering INX api route failed: %s", err)
		}
		cancelRegister()
		Component.LogInfof("Registered INX api route: http://localhost:14265%s", APIRoute)

		Component.LogInfo("Starting API server ... done")
		<-ctx.Done()
		Component.LogInfo("Stopping API ...")

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, routeName); err != nil {
			Component.LogWarnf("Unregistering INX api route failed: %s", err)
		}

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := e.Shutdown(shutdownCtx); err != nil {
			Component.LogWarn(err)
		}

		Component.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopRestAPI); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

func LoadEnvVariables() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
