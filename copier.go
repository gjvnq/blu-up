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
	Hash   string
}

var CopierCh chan CopyOrder
var CopierDoneCh chan bool
var BackupToFolder string
var BackupFromFolder string
var BackupVolUUID string
var BackupVolName string

func AddToCopier(origin, hash string, size int64) {
	order := CopyOrder{}
	order.Size = size
	order.Origin = origin
	order.Dest = BackupToFolder + "/" + Hash2Path(hash)
	order.Hash = hash
	Log.Debug("Added to CoperCh: " + origin)
	CopierCh <- order
}

func copier_consumer() {
	// Do NOT run more than one of this
	for {
		order, more := <-CopierCh
		if !more {
			Log.Notice("Finished copying blobs to " + BackupToFolder)
			CopierDoneCh <- true
			return
		}
		err := copier_main(order)
		if err != nil {
			Log.ErrorF(err.Error())
			continue
		}
		// Verify file size
		info, err := os.Lstat(order.Dest)
		if err != nil {
			Log.ErrorF("Failed to get file size for '%s': %s ", order.Dest, err.Error())
			continue
		}
		if info.Size() != order.Size {
			Log.ErrorF("Original file size (%d bytes) is different from copied file size (%d bytes) for file %s", order.Size, info.Size(), order.Dest)
			continue
		}
		//  Double check everything
		hash, size_hashed, err := hash_file(order.Dest)
		if err != nil {
			Log.ErrorF("Failed to hash file '%s': %s", order.Dest, err.Error())
			continue
		}
		if order.Size != size_hashed {
			Log.WarningF("Oficial blob size (%d bytes) is different from the size hashed (%d bytes)", order.Size, size_hashed)
			continue
		}
		if order.Hash != hash {
			Log.WarningF("Oficial blob hash does not match copied file hash")
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
	return copy_file(order.Origin, order.Dest, order.Size)
}

func copy_file(from, to string, expected_size int64) error {
	// Open source file for reading
	fptr_in, err := os.Open(from)
	defer fptr_in.Close()
	if err != nil {
		Log.WarningF("Failed to open '%s' for reading: %s", from, err.Error())
		return err
	}
	// Open destination for writing
	fptr_out, err := os.Create(to)
	defer fptr_out.Close()
	if err != nil {
		Log.WarningF("Failed to open '%s' for writing: %s", to, err.Error())
		return err
	}
	// Actually copy the file
	size, err := io.Copy(fptr_out, fptr_in)
	if err != nil {
		Log.WarningF("Failed to copy '%s' to '%s': %s", from, to, err.Error())
		return err
	}
	if size != expected_size && expected_size >= 0 {
		Log.WarningF("File size reported by os.Lstat (%d bytes) is different from the size copied (%d bytes) for file %s", expected_size, size, to)
		return errors.New("file size does not match number of hashed bytes")
	}
	return nil
}
