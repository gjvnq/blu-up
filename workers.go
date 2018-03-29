package main

import "io/ioutil"

var INodesToSaveCh chan INode
var PathsToScanCh chan string
var FinishedSavingCh chan bool

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
			scanner_producer(full_path_child, false)
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
		Log.Debug(node.Hash)
	}
}
