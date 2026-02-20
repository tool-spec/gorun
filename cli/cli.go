package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hydrocode-de/gorun/internal/auth"
	"github.com/hydrocode-de/gorun/internal/cache"
	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/sql"
	"github.com/hydrocode-de/gorun/version"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const banner = `
   _____       _____             
  / ____|     |  __ \            
 | |  __  ___ | |__) |     _ __   
 | | |_ |/ _ \|  _  / | | |  _ \  
 | |__| | (_) | | \ \ |_| | | | | 
  \_____|\___/|_|  \_\__,_|_| |_| 
`

var debug bool
var rootCmd = &cobra.Command{
	Use:   "gorun",
	Short: "GoRun operates tool-spec compliant research tools",
	Long: banner + `
GoRun is a CLI tool that operates tool-spec compliant research tools.

The tool specification is available at https://voforwater.github.io/tool-spec/
You ran gorun without a command. Please refer to the section below to learn
about all available commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(0)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initApplicationConfig)

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().String("path", "", "the path to use as the gorun base directory")
	rootCmd.PersistentFlags().String("db_path", "", "the path to use as the database file")

	rootCmd.PersistentFlags().BoolP("version", "v", false, "print the version number of gorun")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Println(version.Version)
			os.Exit(0)
		}
	}

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("path", rootCmd.PersistentFlags().Lookup("path"))
	viper.BindPFlag("db_path", rootCmd.PersistentFlags().Lookup("db_path"))
}

func initApplicationConfig() {
	// Load .env file first
	godotenv.Load()

	viper.SetEnvPrefix("gorun")
	viper.AutomaticEnv()

	viper.SetDefault("port", 8080)
	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("no_auth", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("path", path.Join(os.Getenv("HOME"), ".gorun"))
	viper.SetDefault("db_path", path.Join(viper.GetString("path"), "gorun.db"))
	viper.SetDefault("mount_path", path.Join(viper.GetString("path"), "mounts"))
	viper.SetDefault("temp_path", path.Join(os.TempDir(), "gorun"))
	viper.SetDefault("max_upload_size", 1024*1024*1024*2) // 2GB
	viper.SetDefault("max_temp_age", 12*time.Hour)
	viper.SetDefault("secret", "")
	viper.SetDefault("mcp.enabled", false)
	viper.SetDefault("mcp.transport", "stdio")
	viper.SetDefault("mcp.http.addr", "127.0.0.1:8091")
	viper.SetDefault("mcp.http.auth_required", true)
	viper.SetDefault("mcp.http.insecure_no_auth", false)

	c := &cache.Cache{}
	c.Reset()
	viper.Set("cache", c)

	err := os.MkdirAll(viper.GetString("path"), 0755)
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to create GorunBasePath directory: %w", err))
	}

	// Ensure the database directory exists
	dbDir := path.Dir(viper.GetString("db_path"))
	err = os.MkdirAll(dbDir, 0755)
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to create database directory: %w", err))
	}

	// Ensure the mount directory exists
	err = os.MkdirAll(viper.GetString("mount_path"), 0755)
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to create mount directory: %w", err))
	}

	// Ensure the temp directory exists
	err = os.MkdirAll(viper.GetString("temp_path"), 0755)
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to create temp directory: %w", err))
	}

	// Initialize the database driver
	drv, err := sql.CreateDB(viper.GetString("db_path"))
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to create database driver: %w", err))
	}
	dbQueries := db.New(drv)
	viper.Set("db", dbQueries)

	// validate the config
	cobra.CheckErr(validateConfig())

	// Print debug info if enabled
	printViperState()
}

func validateConfig() error {
	if viper.GetInt("port") == 0 {
		return fmt.Errorf("port is required")
	}

	// check if it is a valid sqlite3 connection string
	dbPath := viper.GetString("db_path")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("the database file %s does not exist", dbPath)
	}

	if viper.GetString("secret") == "" {
		return fmt.Errorf("the secret is required")
	}

	//make sure the AdminCredentials do exist
	ctx := context.Background()
	if _, err := auth.GetAdminCredentials(ctx); err != nil {
		if _, err := auth.CreateAdminCredentials(ctx); err != nil {
			return fmt.Errorf("failed to create admin credentials: %w", err)
		}
	}

	return nil
}

func printViperState() {
	if !viper.GetBool("debug") {
		return
	}

	fmt.Println("\nViper Configuration State:")
	fmt.Println("-------------------------")
	for _, key := range viper.AllKeys() {
		fmt.Printf("%s: %v\n", key, viper.Get(key))
	}
	fmt.Println("-------------------------")
}
