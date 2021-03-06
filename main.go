package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/gjvnq/go-logger"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var Log *logger.Logger
var FlagUUID string
var DBPath string
var DB *sql.DB
var FlagDebug bool
var FlagFix bool
var SigCh chan os.Signal

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

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backups a folder",
	Args:  cobra.NoArgs,
	Run:   backup,
}

func backup(cmd *cobra.Command, args []string) {
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
	BackupFromFolder, _ = filepath.Abs(BackupFromFolder)
	BackupToFolder, _ = filepath.Abs(BackupToFolder)
	if BackupToFolder == BackupFromFolder {
		Log.FatalF("Backup origin ('%s') and destination ('%s') cannot be equal", BackupFromFolder, BackupToFolder)
		return
	}
	vol, err := LoadVol(BackupVolUUID)
	if err != nil {
		Log.FatalF("Failed to load volume %s", BackupVolUUID)
	}
	if vol.UUID == "" {
		Log.FatalF("Volume not found %s", BackupVolUUID)
	}
	BackupVolUUID = vol.UUID
	BackupVolName = vol.Name
	// Start workers
	go inode_scanner_producer(BackupFromFolder, true)
	go inode_scanner_consumer()
	go inode_saver_consumer()
	go copier_consumer()
	Log.Info("Started backup")
	<-FinishedSavingCh
	<-CopierDoneCh
	delete_marked()
	Log.NoticeF("Finished backup from '%s' to '%s' (volume UUID %s)", BackupFromFolder, BackupToFolder, BackupVolUUID)
}

var verifyCmd = &cobra.Command{
	Use:   "verify [uuid] [folder]",
	Short: "Verifies if the backup blobs are fine and repairs them (if requested)",
	Args:  cobra.NoArgs,
	Run:   verify,
}

func verify(cmd *cobra.Command, args []string) {
	// Load DB
	LoadDB(args)
	defer DB.Close()
	// Set up channels
	BlobsToVerifyCh = make(chan VerifyOrder, 128)
	BlobsToRepairCh = make(chan VerifyOrder, 128)
	VerifierWG = &sync.WaitGroup{}

	// Set a few variables
	BackupToFolder, _ = filepath.Abs(BackupToFolder)
	vol, err := LoadVol(BackupVolUUID)
	if err != nil {
		Log.FatalF("Failed to load volume %s", BackupVolUUID)
	}
	if vol.UUID == "" {
		Log.FatalF("Volume not found %s", BackupVolUUID)
	}
	BackupVolUUID = vol.UUID
	BackupVolName = vol.Name
	// Start workers
	VerifierWG.Add(2)
	go verifier_producer()
	go verifier_consumer(FlagFix)
	if FlagFix {
		VerifierWG.Add(1)
		go verifier_fixer()
	}
	if FlagFix {
		Log.Info("Started verification and repair")
	} else {
		Log.Info("Started verification")
	}
	VerifierWG.Wait()
	Log.Notice("Verification complete")
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
	if DBPath == "" {
		DBPath = args[0]
	}
	DBPath, err = filepath.Abs(DBPath)
	if err != nil {
		Log.Fatal(err)
	}
	if !strings.HasSuffix(DBPath, ".sqlite") {
		Log.Fatal("db path must end with .sqlite")
	}
	DB, err = sql.Open("sqlite3", DBPath)
	if err != nil {
		Log.Fatal(err)
	}
	_, err = DB.Exec(CREATE_DB_SQL)
	if err != nil {
		Log.Fatal(err)
	}
}

func cleanup() {
	for {
		<-SigCh
		BeforeFatal()
		os.Exit(1)
	}
}

func BeforeFatal() {
	delete_marked()
}

func main() {
	var err error
	Log, err = logger.New("main", 1, os.Stdout)
	if err != nil {
		panic(err) // Check for error
	}

	// Capture ctrl+c
	SigCh = make(chan os.Signal, 10)
	signal.Notify(SigCh, os.Interrupt, syscall.SIGTERM)
	go cleanup()

	rootCmd.PersistentFlags().BoolVarP(&FlagDebug, "debug", "", false, "show debug info")
	rootCmd.PersistentFlags().StringVarP(&DBPath, "db", "", "", "set the database path")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	volAddCmd.Flags().StringVarP(&FlagUUID, "uuid", "", "", "Force specific UUID for new volume instead of generating a new one")
	volCmd.AddCommand(volAddCmd)
	volCmd.AddCommand(volRmCmd)
	volCmd.AddCommand(volLsCmd)
	rootCmd.AddCommand(volCmd)
	backupCmd.Flags().StringVarP(&BackupFromFolder, "from", "f", "", "path to folder to backup")
	backupCmd.Flags().StringVarP(&BackupToFolder, "to", "t", "", "path to folder to save blobs")
	backupCmd.Flags().StringVarP(&BackupVolUUID, "vol", "v", "", "volume uuid or name")
	backupCmd.MarkFlagRequired("db")
	backupCmd.MarkFlagRequired("from")
	backupCmd.MarkFlagRequired("to")
	backupCmd.MarkFlagRequired("vol")
	rootCmd.AddCommand(backupCmd)
	verifyCmd.Flags().StringVarP(&BackupToFolder, "to", "t", "", "path to folder to save blobs")
	verifyCmd.Flags().StringVarP(&BackupVolUUID, "vol", "v", "", "volume uuid or name")
	verifyCmd.Flags().BoolVarP(&FlagFix, "fix", "f", false, "attempt to fix wrong or missing blobs")
	verifyCmd.MarkFlagRequired("db")
	verifyCmd.MarkFlagRequired("to")
	verifyCmd.MarkFlagRequired("vol")
	rootCmd.AddCommand(verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
