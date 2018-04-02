package main

import (
	"os"
	"path/filepath"
	"sync"
)

var BlobsToRepairCh chan VerifyOrder
var BlobsToVerifyCh chan VerifyOrder
var VerifierWG *sync.WaitGroup

type VerifyOrder struct {
	Hash string
	Size int64
	Path string
}

func verifier_producer() {
	defer VerifierWG.Done()
	defer close(BlobsToVerifyCh)

	// Query
	rows, err := DB.Query("SELECT `hash`, `size` FROM `blobs` WHERE `volume_uuid` = ?;", BackupVolUUID)
	if err != nil {
		Log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		var size int64
		err := rows.Scan(&hash, &size)
		if err != nil {
			Log.Fatal(err)
		}
		order := VerifyOrder{}
		order.Hash = hash
		order.Size = size
		order.Path = filepath.Join(BackupToFolder, Hash2Path(hash))
		Log.DebugF("Adding '%s' for verification", order.Path)
		BlobsToVerifyCh <- order
	}
}

func verifier_consumer(repair bool) {
	defer VerifierWG.Done()
	for {
		order, more := <-BlobsToVerifyCh
		if !more {
			if repair {
				close(BlobsToRepairCh)
			}
			Log.Info("No more blobs to verify")
			return
		}
		verifier_consumer_main(order, repair)
	}
}

func add_blob_to_repair(order VerifyOrder, repair bool) {
	if repair {
		BlobsToRepairCh <- order
	}
}

func verifier_consumer_main(order VerifyOrder, repair bool) {
	level := "ERROR"
	if repair {
		level = "WARNING"
	}

	Log.DebugF("Verifing '%s'", order.Path)
	info, err := os.Lstat(order.Path)
	if err != nil {
		Log.LogF(level, "Failed to get file size for '%s': %s ", order.Path, err.Error())
		add_blob_to_repair(order, repair)
		return
	}
	// Check size
	if info.Size() != order.Size {
		Log.LogF(level, "Real file size (%d bytes) is different from blob file size (%d bytes) for file %s", info.Size(), order.Size, order.Path)
		add_blob_to_repair(order, repair)
		return
	}
	// Check hash
	hash, size_hashed, err := hash_file(order.Path)
	if err != nil {
		Log.LogF(level, "Failed to hash file '%s': %s", order.Path, err.Error())
		add_blob_to_repair(order, repair)
		return
	}
	if order.Size != size_hashed {
		Log.LogF(level, "Oficial blob size (%d bytes) is different from the size hashed (%d bytes)", order.Size, size_hashed)
		add_blob_to_repair(order, repair)
		return
	}
	if order.Hash != hash {
		Log.LogF(level, "Oficial blob hash does not match copied file hash for '%s'", order.Path)
		add_blob_to_repair(order, repair)
		return
	}
}

func verifier_fixer() {
	Log.Debug("Started verifier_fixer")
	defer VerifierWG.Done()
	for {
		order, more := <-BlobsToRepairCh
		if !more {
			Log.Info("No more blobs to repair")
			return
		}
		verifier_fixer_main(order)
	}
}

func verifier_fixer_main(order VerifyOrder) {
	Log.InfoF("Looking for files to repair blob %s", order.Hash)
	// Look for inodes that might still have the same blob
	rows, err := DB.Query("SELECT `original_path` FROM `inodes` WHERE `hash`= ? AND `type` = ? GROUP BY `original_path`;", order.Hash, INODE_TYPE_FILE)
	if err != nil {
		Log.Error(err)
	}
	paths := make([]string, 0)
	for rows.Next() {
		var path string
		err := rows.Scan(&path)
		if err != nil {
			Log.Error(err)
		}
		paths = append(paths, path)
	}
	rows.Close()
	// Attempt to use those inodes to repair blob
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			Log.DebugF("Failed to get file size for '%s': %s ", path, err.Error())
			continue
		}
		// Check size
		if info.Size() != order.Size {
			Log.DebugF("Real file size (%d bytes) is different from blob file size (%d bytes) for file %s", info.Size(), order.Size, path)
			continue
		}
		// Check hash
		hash, size_hashed, err := hash_file(path)
		if err != nil {
			Log.DebugF("Failed to hash file '%s': %s", path, err.Error())
			continue
		}
		if order.Size != size_hashed {
			Log.DebugF("Oficial blob size (%d bytes) is different from the size hashed (%d bytes)", order.Size, size_hashed)
			continue
		}
		if order.Hash != hash {
			Log.DebugF("Oficial blob hash does not match copied file hash for '%s'", path)
			continue
		}
		// File is usable, let's copy it
		err = copy_file(path, order.Path, order.Size)
		if err == nil {
			Log.NoticeF("Successfully repaired blob '%s' using file '%s'", order.Hash, path)
			return
		}
	}
	Log.ErrorF("Failed to repair blob '%s'", order.Hash)
}
