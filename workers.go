package main

import (
	"io/ioutil"
	"os"
	"sync"
)

var INodesToSaveCh chan INode
var PathsToScanCh chan string
var FinishedSavingCh chan bool
var IgnoreFolders []string = []string{".git", ".cvs", ".svn", ".cache"}
var SpecialFoldersToPack []string = []string{".git", ".svn", ".hg"}
var MarkedForDeletion []string
var MarkedForDeletionLock *sync.Mutex

func ContainsStr(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func saver_consumer() {
	for {
		inode, more := <-INodesToSaveCh
		if !more {
			Log.Notice("Finished saving inodes to database")
			close(CopierCh)
			FinishedSavingCh <- true
			return
		}
		err := inode.Save()
		if err != nil {
			Log.Warning(err)
			continue
		}
		// Check for blob
		if inode.Hash == "" {
			continue
		}
		blob, err := LoadBlob(inode.Hash)
		if err != nil {
			Log.Warning(err)
		}
		if blob.Hash != "" {
			Log.DebugF("Found blob for '%s' on volume %s", inode.OriginalPath, blob.VolUUID)
		} else {
			blob.Hash = inode.Hash
			blob.Size = inode.Size
			blob.VolUUID = BackupVolUUID
			Log.DebugF("Blob for '%s' has not been copied yet", inode.HackPath)
			err = blob.Save()
			if err != nil {
				Log.Warning(err)
				continue
			}
			AddToCopier(inode.HackPath, inode.Hash, blob.Size)
		}
	}
}

// Recursivelly lists the filesystem in order to list what inodes will be scanned. DO NOT run more than one goroutine for this
func scanner_producer(root string, is_root bool) {
	if is_root {
		Log.NoticeF("Started looking for files to backup on '%s'", root)
	}
	children, err := ioutil.ReadDir(root)
	if err != nil {
		Log.Warning(err)
	}
	for _, child := range children {
		full_path_child := root + "/" + child.Name()
		PathsToScanCh <- full_path_child
		if child.IsDir() {
			if !ContainsStr(IgnoreFolders, child.Name()) && !ContainsStr(SpecialFoldersToPack, child.Name()) {
				scanner_producer(full_path_child, false)
			}
		}
	}

	if is_root {
		Log.Info("Finished paths to scan")
		close(PathsToScanCh)
	}
}

func scanner_consumer() {
	for {
		path, more := <-PathsToScanCh
		if !more {
			Log.Info("Closing INodesToSaveCh...")
			close(INodesToSaveCh)
			return
		}
		node, err := NewINodeFromFile(path)
		if err != nil {
			Log.Warning(node.OriginalPath, err)
		}
	}
}

func delete_marked() {
	var err error

	MarkedForDeletionLock.Lock()
	Log.Notice("Deleting temporary files created during backup")
	defer MarkedForDeletionLock.Unlock()
	for _, path := range MarkedForDeletion {
		err = os.Remove(path)
		if err != nil {
			Log.Warning("Failed to delete '" + path + "': " + err.Error())
		} else {
			Log.Info("Deleted " + path)
		}
	}
	Log.Notice("Deleted all temporary files created during backup")
}
