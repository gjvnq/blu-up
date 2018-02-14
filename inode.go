package main

const INODE_TYPE_FILE = "f"
const INODE_TYPE_DIRECTORY = "d"
const INODE_TYPE_SYMBOLIC_LINK = "l"

type INode struct {
	Id           int64
	Type         string
	OriginalPath string
	MimeType     string
	Hash         string // SHA3-512 (if it is a link, the hash value will be of the referenced file)
	Target       string // Used only for links
	Size         int64  // In bytes
	User         string
	Group        string
	Chmod        int
}
