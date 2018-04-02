package main

import (
	"fmt"

	uuid "github.com/gjvnq/go.uuid"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

type Vol struct {
	UUID string `json:uuid`
	Name string `json:name`
	Desc string `json:desc`
}

func NewVol() Vol {
	vol := Vol{}
	vol.UUID = uuid.NewV4().String()
	return vol
}

func LoadVol(name_or_uuid string) (Vol, error) {
	vol := Vol{}
	err := DB.QueryRow("SELECT `uuid`, `name`, `desc` FROM `volumes` WHERE `uuid` = ? OR `name` = ?;", name_or_uuid, name_or_uuid).Scan(&vol.UUID, &vol.Name, &vol.Desc)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = nil
		} else {
			Log.Warning(err)
		}
	}
	return vol, err
}

var volCmd = &cobra.Command{
	Use:   "vol",
	Short: "Manage volumes",
}

var volAddCmd = &cobra.Command{
	Use:   "add [name] [desc]",
	Short: "Adds a volume to the database",
	Args:  cobra.RangeArgs(1, 2),
	Run:   volAdd,
}

func volAdd(cmd *cobra.Command, args []string) {
	// Load DB
	LoadDB(args)
	defer DB.Close()
	// Query
	vol := NewVol()
	if FlagUUID != "" {
		vol.UUID = FlagUUID
	}
	vol.Name = args[0]
	if len(args) > 1 {
		vol.Desc = args[1]
	}
	_, err := DB.Exec("INSERT INTO `volumes` (`uuid`, `name`, `desc`) VALUES (?, ?, ?);", vol.UUID, vol.Name, vol.Desc)
	if err != nil {
		Log.Fatal(err)
	}
}

var volRmCmd = &cobra.Command{
	Use:   "rm [uuid]",
	Short: "Removes a volume from the database",
	Args:  cobra.ExactArgs(1),
	Run:   volRm,
}

func volRm(cmd *cobra.Command, args []string) {
	// Load DB
	LoadDB(args)
	defer DB.Close()
	// Query
	_, err := DB.Exec("DELETE FROM `volumes` WHERE `uuid` = ?", args[0])
	if err != nil {
		Log.Fatal(err)
	}
}

var volLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Lists the volumes in the database",
	Args:  cobra.NoArgs,
	Run:   volLs,
}

func volLs(cmd *cobra.Command, args []string) {
	flag_empty := true

	// Load DB
	LoadDB(args)
	defer DB.Close()
	// Query
	rows, err := DB.Query("SELECT `uuid`, `name`, `desc` FROM `volumes`;")
	if err != nil {
		Log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		flag_empty = false
		var uuid string
		var name string
		var desc string
		err := rows.Scan(&uuid, &name, &desc)
		if err != nil {
			Log.Fatal(err)
		}
		fmt.Println(uuid, aurora.Bold(name), desc)
	}
	if flag_empty {
		fmt.Println("no volumes in the database")
	}
}
