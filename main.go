package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/krateoplatformops/github-rest-dynamic-controller-plugin/docs"
	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/env"
	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers/repo"
	teamrepo "github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers/teamRepo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	serviceName = "github-plugin"
)

// @title           GitHub Plugin API for Krateo Operator Generator (KOG)
// @version         1.0
// @description     Simple wrapper around GitHub API to provide consisentency of API response for Krateo Operator Generator (KOG)
// @termsOfService  http://swagger.io/terms/

// @contact.name   Krateo Support
// @contact.url    https://krateo.io
// @contact.email  contact@krateoplatformops.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host			localhost:8080
// @BasePath		/
// @schemes 	 	http

// @securityDefinitions.basic  Bearer

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	debugOn := flag.Bool("debug", env.Bool("EVENTSSE_DEBUG", true), "dump verbose output")
	port := flag.Int("port", env.Int("EVENTSSE_PORT", 8080), "port to listen on")
	noColor := flag.Bool("no-color", env.Bool("EVENTSSE_NO_COLOR", false), "disable color output")

	flag.Parse()

	mux := http.NewServeMux()

	// Initialize the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Default level for this log is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debugOn {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: *noColor,
	}).With().Timestamp().Logger()

	opts := handlers.HandlerOptions{
		Log:    &log.Logger,
		Client: http.DefaultClient,
	}

	healthy := int32(0)

	mux.Handle("GET /repository/{owner}/{repo}/collaborators/{username}/permission", repo.GetRepo(opts))
	mux.Handle("GET /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamrepo.GetTeamRepo(opts))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGHUP,
		syscall.SIGQUIT,
	}...)
	defer stop()

	go func() {
		atomic.StoreInt32(&healthy, 1)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("could not listen on %s", server.Addr)
		}
	}()

	// Listen for the interrupt signal.
	log.Info().Msgf("server is ready to handle requests at %s", server.Addr)
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("server is shutting down gracefully, press Ctrl+C again to force")
	atomic.StoreInt32(&healthy, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server gracefully stopped")
}
