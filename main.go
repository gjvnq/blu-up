package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gjvnq/go-logger"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var Log *logger.Logger
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
	Use:   "run [db path] [folder to backup] [volume uuid] [volume path]",
	Short: "Backups a folder",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		// Load DB
		LoadDB(args)
		defer DB.Close()
		// Set up channels
		PathsToScanCh = make(chan string, 128)
		INodesToSaveCh = make(chan INode, 2048)
		FinishedSavingCh = make(chan bool)

		// Set a few variables
		backup_path, _ := filepath.Abs(args[1])
		// vol_uuid := args[2]
		// target_path := args[3]
		// fmt.Println(NewINodeFromFile(backup_path))
		go scanner_producer(backup_path, true)
		go scanner_consumer()
		go saver_consumer()
		Log.Info("waiting...")
		<-FinishedSavingCh
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
	},
}

func LoadDB(args []string) {
	var err error
	DBPath, err = filepath.Abs(args[0])
	if err != nil {
		Log.Fatal(err)
	}
	if !strings.HasSuffix(DBPath, ".sqlite") {
		Log.Fatal("db path must end with .sqlite")
	}
	DB, err = sql.Open("sqlite3", DBPath)
	DB.SetMaxOpenConns(1)
	if err != nil {
		Log.Fatal(err)
	}
	_, err = DB.Exec(CREATE_DB_SQL)
	if err != nil {
		Log.Fatal(err)
	}
}

func main() {
	var err error
	Log, err = logger.New("main", 1, os.Stdout)
	if err != nil {
		panic(err) // Check for error
	}

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
