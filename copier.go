package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type CopyOrder struct {
	Origin string
	Dest   string
	Size   int64
}

var CopierCh chan CopyOrder
var CopierDoneCh chan bool
var BackupToFolder string
var BackupFromFolder string
var BackupVolUUID string

func AddToCopier(origin, hash string, size int64) {
	order := CopyOrder{}
	order.Size = size
	order.Origin = origin
	order.Dest = BackupToFolder + "/" + Hash2Path(hash)
	Log.Debug("Added to CoperCh: " + origin)
	CopierCh <- order
}

func copier_consumer() {
	for {
		order, more := <-CopierCh
		if !more {
			Log.Info("Finished copying inodes to " + BackupToFolder)
			CopierDoneCh <- true
			return
		}
		err := copier_main(order)
		if err != nil {
			Log.Warning(err)
			continue
		}
		// Verify file size
		info, err := os.Lstat(order.Dest)
		if err != nil {
			Log.WarningF("copier_consumer(path = '%s') (os.Lstat): %s ", order.Dest, err)
			continue
		}
		if info.Size() != order.Size {
			Log.WarningF("original file size (%d bytes) is different from copied file size (%d bytes) for file %s", order.Size, info.Size(), order.Dest)
			continue
		}
	}
}

func copier_main(order CopyOrder) error {
	// Ensure folder exists
	dir := filepath.Dir(order.Dest)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		Log.WarningF("Failed to create '%s' and parent folders: %s", dir, err.Error())
		return err
	}
	// Open source file for reading
	fptr_in, err := os.Open(order.Origin)
	defer fptr_in.Close()
	if err != nil {
		Log.WarningF("Failed to open '%s' for reading: %s", order.Origin, err.Error())
		return err
	}
	// Open destination for writing
	fptr_out, err := os.Create(order.Dest)
	defer fptr_out.Close()
	if err != nil {
		Log.WarningF("Failed to open '%s' for writing: %s", order.Dest, err.Error())
		return err
	}
	// Actually copy the file
	size, err := io.Copy(fptr_out, fptr_in)
	if err != nil {
		Log.WarningF("Failed to copy '%s' to '%s': %s", order.Origin, order.Dest, err.Error())
		return err
	}
	if size != order.Size {
		Log.WarningF("File size reported by os.Lstat (%d bytes) is different from the size copied (%d bytes) for file %s", order.Size, size, order.Dest)
		return errors.New("file size does not match number of hashed bytes")
	}
	return nil
}
