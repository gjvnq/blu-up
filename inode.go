package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gjvnq/go.uuid"
	"golang.org/x/crypto/sha3"
)

const INODE_TYPE_FILE = "f"
const INODE_TYPE_DIRECTORY = "d"
const INODE_TYPE_SYMBOLIC_LINK = "l"

type INode struct {
	UUID         string    `json:uuid`
	Type         string    `json:type`
	Hash         string    `json:hash` // If it is a link, this will be null
	OriginalPath string    `json:original_path`
	TargetPath   string    `json:target_path` // Used only for links
	Size         int64     `json:size`        // In bytes
	User         string    `json:user`
	Group        string    `json:group`
	Mode         string    `json:mode`
	ModTime      time.Time `json:mod_time`
	ScanTime     time.Time `json:scan_time`
}

const ERR_INVALID_INODE_TYPE = "invalid inode type (ex: sockets)"

func SaveInode(inode INode) {
	_, err := DB.Exec("INSERT INTO `inodes` (`uuid`, `type`, `hash`, `original_path`, `target_path`, `size`, `user`, `group`, `mode`, `mod_time`, `scan_time`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);", inode.UUID, inode.Type, inode.Hash, inode.OriginalPath, inode.TargetPath, inode.Size, inode.User, inode.Group, inode.Mode, inode.ModTime.Unix(), inode.ScanTime.Unix())
	if err != nil {
		Log.Warning(err)
	} else {
		Log.Debug("Saved " + inode.UUID + " on " + inode.OriginalPath + " to the database")
	}
}

func NewINodeFromFile(path string) (*INode, error) {
	node := &INode{}
	err := node.FromFile(path)
	INodesToSaveCh <- *node
	return node, err
}

func (node *INode) FromFile(path string) error {
	var err error

	// Generate UUID and set scan time
	node.UUID = uuid.NewV4().String()
	node.ScanTime = time.Now()

	// Get absolute path
	node.OriginalPath, err = filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (filepath.Abs): %s ", path, err)
		return err
	}

	// Get file info
	info, err := os.Lstat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (os.Lstat): %s ", path, err)
		return err
	}
	node.ModTime = info.ModTime()

	// Get user and group
	if info.Sys() != nil {
		uid := fmt.Sprintf("%d", info.Sys().(*syscall.Stat_t).Uid)
		gid := fmt.Sprintf("%d", info.Sys().(*syscall.Stat_t).Gid)
		u, _ := user.LookupId(uid)
		g, _ := user.LookupGroupId(gid)
		node.User = u.Username
		node.Group = g.Name
	}

	// Get file mode/type
	node.Mode = info.Mode().String()
	if info.Mode().IsDir() {
		node.Type = INODE_TYPE_DIRECTORY
		// Directories have no hash
		node.Hash = ""
		return nil
	} else if info.Mode().IsRegular() {
		node.Type = INODE_TYPE_FILE
	} else if info.Mode()&os.ModeSymlink != 0 {
		node.Type = INODE_TYPE_SYMBOLIC_LINK
		// Links have no hash, but have a Target Path
		node.Hash = ""
		node.TargetPath, err = os.Readlink(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (os.Readlink): %s \n", path, err)
			return err
		}
		return nil
	} else {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (invalid inode type, ex: sockets): %s \n", path, node.Mode)
		return errors.New(ERR_INVALID_INODE_TYPE)
	}

	// Open file
	fptr, err := os.Open(path)
	defer fptr.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (opening file): %s \n", path, err)
		return err
	}

	// Hash the file
	hasher := sha3.New512()
	if node.Size, err = io.Copy(hasher, fptr); err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (hashing file): %s \n", path, err)
		return err
	}

	// Store the hash
	node.Hash = "SHA3-512:" + hex.EncodeToString(hasher.Sum(nil))

	return nil
}
