package main

import (
	"io/ioutil"
	"os"
)

var INodesToSaveCh chan INode
var PathsToScanCh chan string
var FinishedSavingCh chan bool
var IgnoreFolders []string = []string{".git", ".cvs", ".svn", ".cache"}
var SpecialFoldersToPack []string = []string{".git", ".svn", ".hg"}
var MarkedForDeletion []string

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
			Log.Info("Finished saving INodes to database")
			FinishedSavingCh <- true
			return
		}
		SaveInode(inode)
	}
}

// Recursivelly lists the filesystem in order to list what inodes will be scanned. DO NOT run more than one goroutine for this
func scanner_producer(root string, is_root bool) {
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
			Log.Warning(node, err)
		}
	}
}

func delete_marked() {
	var err error

	Log.Info("Deleting temporary files created during backup")
	for _, path := range MarkedForDeletion {
		err = os.Remove(path)
		if err != nil {
			Log.Warning("Failed to delete '" + path + "': " + err.Error())
		} else {
			Log.Info("Deleted " + path)
		}
	}
}
