package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var FlagUUID string
var DBPath string
var DB *sql.DB

const VERSION = "v0.0.1"

var rootCmd = &cobra.Command{
	Use:   "blu-up [command]",
	Short: "blu-up a simple backup tool",
	Long:  "A hash based backup tool capable of multiple volumes, links and deduplication. https://github.com/gjvnq/blu-up",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of blu-up",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

var runCmd = &cobra.Command{
	Use:   "run [db path] [folder to backup] [volume name] [volume path]",
	Short: "Backups a folder",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v0.0.1")
	},
}

var initCmd = &cobra.Command{
	Use:   "init [db path]",
	Short: "Starts an empty backup database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load DB
		LoadDB(args)
		defer DB.Close()
		// Run commands
		_, err := DB.Exec(CREATE_DB_SQL)
		CheckEnd(err)
	},
}

func LoadDB(args []string) {
	var err error
	DBPath, err = filepath.Abs(args[0])
	CheckEnd(err)
	if !strings.HasSuffix(DBPath, ".sqlite") {
		CheckEnd("db path must end with .sqlite")
	}
	DB, err = sql.Open("sqlite3", DBPath)
	CheckEnd(err)
	// fmt.Println("Opened", DBPath)
}

func CheckEnd(err interface{}) {
	if err == nil {
		return
	}
	fmt.Println(aurora.Bold(aurora.Red(err)))
	os.Exit(1)
}

func main() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	volAddCmd.Flags().StringVarP(&FlagUUID, "uuid", "", "", "Force specific UUID for new volume instead of generating a new one")
	volCmd.AddCommand(volAddCmd)
	volCmd.AddCommand(volRmCmd)
	volCmd.AddCommand(volLsCmd)
	rootCmd.AddCommand(volCmd)
	rootCmd.AddCommand(runCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
