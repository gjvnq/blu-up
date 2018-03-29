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

var volCmd = &cobra.Command{
	Use:   "vol",
	Short: "Manage volumes",
}

var volAddCmd = &cobra.Command{
	Use:   "add [db path] [name] [desc]",
	Short: "Adds a volume to the database",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		// Load DB
		LoadDB(args)
		defer DB.Close()
		// Query
		vol := NewVol()
		if FlagUUID != "" {
			vol.UUID = FlagUUID
		}
		vol.Name = args[1]
		if len(args) > 2 {
			vol.Desc = args[2]
		}
		_, err := DB.Exec("INSERT INTO `volumes` (`uuid`, `name`, `desc`) VALUES (?, ?, ?);", vol.UUID, vol.Name, vol.Desc)
		if err != nil {
			Log.Fatal(err)
		}
	},
}

var volRmCmd = &cobra.Command{
	Use:   "rm [db path] [uuid]",
	Short: "Removes a volume from the database",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Load DB
		LoadDB(args)
		defer DB.Close()
		// Query
		_, err := DB.Exec("DELETE FROM `volumes` WHERE `uuid` = ?", args[1])
		if err != nil {
			Log.Fatal(err)
		}
	},
}

var volLsCmd = &cobra.Command{
	Use:   "ls [db path]",
	Short: "Lists the volumes in the database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}
