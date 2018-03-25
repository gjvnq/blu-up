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
	UUID         uuid.UUID `json:uuid`
	Type         string    `json:type`
	OriginalPath string    `json:original_path`
	Hash         string    `json:hash`        // If it is a link, this will be null
	TargetPath   string    `json:target_path` // Used only for links
	Size         int64     `json:size`        // In bytes
	User         string    `json:user`
	Group        string    `json:group`
	Mode         string    `json:mode`
	ModTime      time.Time `json:mod_tile`
}

func (node *INode) FromFile(path string) error {
	var err error

	// Generate UUID
	node.UUID = uuid.NewV4()

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
	} else if info.Mode().IsRegular() {
		node.Type = INODE_TYPE_FILE
	} else if info.Mode()&os.ModeSymlink != 0 {
		node.Type = INODE_TYPE_SYMBOLIC_LINK
		// Links have no hash, but have a Target Path
		node.Hash = ""
		node.TargetPath, err = os.Readlink(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (os.Readlink): %s ", path, err)
			return err
		}
		return nil
	} else {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (invalid inode type, ex: sockets): %s ", path, node.Mode)
		return errors.New("invalid inode type (ex: sockets)")
	}

	// Open file
	fptr, err := os.Open(path)
	defer fptr.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (opening file): %s ", path, err)
		return err
	}

	// Hash the file
	hasher := sha3.New512()
	if node.Size, err = io.Copy(hasher, fptr); err != nil {
		fmt.Fprintf(os.Stderr, "FromFile(path = '%s') (hashing file): %s ", path, err)
		return err
	}

	// Store the hash
	node.Hash = "SHA3-512:" + hex.EncodeToString(hasher.Sum(nil))

	return nil
}
