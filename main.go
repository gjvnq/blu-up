package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gjvnq/go-logger"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var Log *logger.Logger
var FlagUUID string
var DBPath string
var DB *sql.DB
var FlagDebug bool

const VERSION = "v0.0.1"

var rootCmd = &cobra.Command{
	Use:   "blu-up [command]",
	Short: "blu-up a simple backup tool",
	Long:  "A hash based backup tool capable of multiple volumes, links and deduplication. https://github.com/gjvnq/blu-up",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Log.Worker.DisabledLevels["DEBUG"] = !FlagDebug
	},
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
		CopierCh = make(chan CopyOrder, 64)
		PathsToScanCh = make(chan string, 128)
		INodesToSaveCh = make(chan INode, 2048)
		FinishedSavingCh = make(chan bool)
		CopierDoneCh = make(chan bool)
		MarkedForDeletion = make([]string, 0)
		MarkedForDeletionLock = &sync.Mutex{}

		// Set a few variables
		BackupFromFolder, _ = filepath.Abs(args[1])
		BackupVolUUID = args[2]
		BackupToFolder = args[3]
		// Start workers
		go scanner_producer(BackupFromFolder, true)
		go scanner_consumer()
		go saver_consumer()
		go copier_consumer()
		<-FinishedSavingCh
		<-CopierDoneCh
		delete_marked()
		Log.NoticeF("Finished backup from '%s' to '%s' (volume UUID %s)", BackupFromFolder, BackupToFolder, BackupVolUUID)
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

	rootCmd.PersistentFlags().BoolVarP(&FlagDebug, "debug", "", false, "show debug info")
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
