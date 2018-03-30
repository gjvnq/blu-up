package main

import (
	"strings"
	"time"
)

type Blob struct {
	Hash       string    `json:hash`
	Size       int64     `json:size`
	VolUUID    string    `json:volume_uuid`
	FirstAdded time.Time `json:first_added`
}

func Hash2Path(src_hash string) string {
	parts := strings.Split(src_hash, ":")
	alg := parts[0]
	hash := parts[1]
	return alg + "/" + hash[0:3] + "/" + hash[3:6] + "/" + hash
}

func LoadBlob(hash string) (Blob, error) {
	blob := Blob{}
	err := DB.QueryRow("SELECT `hash`, `size`, `volume_uuid` FROM `blobs` WHERE `hash`= ?", hash).Scan(&blob.Hash, &blob.Size, &blob.VolUUID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = nil
		} else {
			Log.Warning(err)
		}
	}
	return blob, err
}

func (blob *Blob) Save() error {
	blob.FirstAdded = time.Now()
	_, err := DB.Exec("INSERT INTO `blobs` (`hash`, `size`, `volume_uuid`, `first_added`) VALUES (?, ?, ?, ?);", blob.Hash, blob.Size, blob.VolUUID, blob.FirstAdded.Unix())
	if err != nil {
		Log.Warning(err)
	} else {
		Log.DebugF("Saved blob '%s' to database", blob.Hash)
	}
	return err
}
