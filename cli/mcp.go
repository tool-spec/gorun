package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hydrocode-de/gorun/internal/auth"
	"github.com/hydrocode-de/gorun/internal/cache"
	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/mcp"
	"github.com/hydrocode-de/gorun/internal/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the GoRun MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		transport := viper.GetString("mcp.transport")
		httpAddr := viper.GetString("mcp.http.addr")
		authRequired := viper.GetBool("mcp.http.auth_required")
		insecureNoAuth := viper.GetBool("mcp.http.insecure_no_auth")

		if insecureNoAuth {
			log.Printf("WARNING: MCP HTTP server running with --mcp-http-no-auth")
		}

		startBackgroundTasks(cmd.Context())

		svc := &service.Service{
			DB:    viper.Get("db").(*db.Queries),
			Cache: viper.Get("cache").(*cache.Cache),
		}
		server := mcp.NewServer(svc, mcp.Config{AuthRequired: authRequired, InsecureNoAuth: insecureNoAuth})

		if _, err := ensureAdminUser(cmd); err != nil {
			cobra.CheckErr(err)
		}

		switch transport {
		case "stdio":
			log.Printf("GoRun MCP server listening on stdio")
			cobra.CheckErr(server.RunStdio(cmd.Context(), os.Stdin, os.Stdout))
		case "http":
			log.Printf("GoRun MCP server listening on http://%s/mcp", httpAddr)
			cobra.CheckErr(http.ListenAndServe(httpAddr, server.HTTPHandler()))
		case "both":
			go func() {
				log.Printf("GoRun MCP server listening on http://%s/mcp", httpAddr)
				if err := http.ListenAndServe(httpAddr, server.HTTPHandler()); err != nil {
					log.Printf("MCP HTTP server error: %v", err)
				}
			}()
			log.Printf("GoRun MCP server listening on stdio")
			cobra.CheckErr(server.RunStdio(cmd.Context(), os.Stdin, os.Stdout))
		default:
			cobra.CheckErr(fmt.Errorf("invalid mcp transport %q (expected stdio|http|both)", transport))
		}
	},
}

func ensureAdminUser(cmd *cobra.Command) (string, error) {
	credentials, err := auth.GetAdminCredentials(cmd.Context())
	if err != nil {
		credentials, err = auth.CreateAdminCredentials(cmd.Context())
		if err != nil {
			return "", err
		}
	}
	return credentials.UserID, nil
}

func init() {
	mcpServeCmd.Flags().String("transport", "", "MCP transport to use: stdio|http|both")
	mcpServeCmd.Flags().String("mcp-http-addr", "", "MCP HTTP bind address")
	mcpServeCmd.Flags().Bool("mcp-http-auth-required", true, "Require Bearer auth for MCP HTTP requests")
	mcpServeCmd.Flags().Bool("mcp-http-no-auth", false, "Disable auth for MCP HTTP (unsafe, local dev only)")

	_ = viper.BindPFlag("mcp.transport", mcpServeCmd.Flags().Lookup("transport"))
	_ = viper.BindPFlag("mcp.http.addr", mcpServeCmd.Flags().Lookup("mcp-http-addr"))
	_ = viper.BindPFlag("mcp.http.auth_required", mcpServeCmd.Flags().Lookup("mcp-http-auth-required"))
	_ = viper.BindPFlag("mcp.http.insecure_no_auth", mcpServeCmd.Flags().Lookup("mcp-http-no-auth"))

	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}
