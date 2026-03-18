package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hydrocode-de/gorun/api"
	"github.com/hydrocode-de/gorun/internal/auth"
	"github.com/hydrocode-de/gorun/internal/cache"
	"github.com/hydrocode-de/gorun/internal/files"
	"github.com/hydrocode-de/gorun/internal/toolImage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var port int
var host string
var noAuth bool
var waitForCache bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the GoRun API server",
	Run: func(cmd *cobra.Command, args []string) {
		serverHost := viper.GetString("host")
		serverPort := viper.GetInt("port")
		serverNoAuth := viper.GetBool("no_auth")

		if serverNoAuth && serverHost != "127.0.0.1" {
			log.Printf("WARNING: You are running the server with no authentication and a non-localhost host. This is not recommended and might expose your server to the public internet.")
		}

		// Start background tasks and optionally wait for cache initialization
		if waitForCache {
			log.Println("Waiting for cache initialization before starting server...")
			startBackgroundTasksAndWait(cmd.Context())
		} else {
			startBackgroundTasks(cmd.Context())
		}

		mux, err := api.CreateServer()
		cobra.CheckErr(err)

		server := api.EnableCORS(mux, "*")
		log.Printf("GoRun server listening on  http://%s:%d\n", serverHost, serverPort)
		log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", serverHost, serverPort), server))
	},
}

func startBackgroundTasks(ctx context.Context) {
	// Initial cache population
	log.Println("Initializing tool cache...")
	cacheInstance := viper.Get("cache").(*cache.Cache)
	_, err := toolImage.ReadAllTools(ctx, cacheInstance, false)
	if err != nil {
		log.Printf("Warning: Failed to initialize tool cache: %v", err)
	} else {
		log.Println("Tool cache initialized successfully")
	}

	startPeriodicTasks(ctx)
}

func startBackgroundTasksAndWait(ctx context.Context) {
	// Initial cache population with waiting
	log.Println("Initializing tool cache...")
	cacheInstance := viper.Get("cache").(*cache.Cache)
	_, err := toolImage.ReadAllTools(ctx, cacheInstance, false)
	if err != nil {
		log.Printf("Warning: Failed to initialize tool cache: %v", err)
	} else {
		log.Println("Tool cache initialized successfully")
	}

	// Wait for cache to be marked as initialized
	for !cacheInstance.IsInitialised() {
		log.Println("Waiting for cache initialization to complete...")
		time.Sleep(time.Second)
	}
	log.Println("Cache initialization completed, starting server...")

	startPeriodicTasks(ctx)
}

func startPeriodicTasks(ctx context.Context) {
	cleanupTicker := time.NewTicker(time.Minute * 5)
	go func() {
		for range cleanupTicker.C {
			log.Println("Running cleanup")
			err := files.Cleanup()
			cobra.CheckErr(err)
		}
	}()

	toolsTicker := time.NewTicker(time.Minute * 5)
	go func() {
		for range toolsTicker.C {
			log.Println("Checking for new tools")
			cacheInstance := viper.Get("cache").(*cache.Cache)
			_, err := toolImage.ReadAllTools(ctx, cacheInstance, false)
			cobra.CheckErr(err)
		}
	}()

	adminTicker := time.NewTicker(time.Minute * 50)
	go func() {
		for range adminTicker.C {
			log.Println("Renewing admin credentials")
			if _, err := auth.GetAdminCredentials(ctx); err != nil {
				log.Printf("Failed to renew admin credentials: %v...\n", err)
			}
		}
	}()
}

func init() {
	serveCmd.Flags().IntVar(&port, "port", 0, "The port to listen on")
	serveCmd.Flags().StringVar(&host, "host", "", "The host to listen on")
	serveCmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable authentication")
	serveCmd.Flags().BoolVar(&waitForCache, "wait-for-cache", false, "Wait for cache initialization before starting the server")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("no_auth", serveCmd.Flags().Lookup("no-auth"))

	rootCmd.AddCommand(serveCmd)
}
