package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"crypto/rand"

	"golang.org/x/crypto/sha3"
)

type BinSeq struct {
	hash     []byte // SHA3-512 of the original file
	size     int64  // in bytes of the original file
	path     string
	end_path string
	aes_key []byte
	enc_hash     []byte // SHA3-512 of the encripted file
	enc_size     int64  // in bytes of the encripted file
	enc_end_path string
}

type EncBinSeq struct {
}

func (bs BinSeq) EndPath() string {
	return bs.enc_end_path
}

func (bs BinSeq) GetKey() string {
	return hex.EncodeToString(hasher.Sum(nil))
}

func (bs *BinSeq) ProcessFile(path string) error {
	bs.path = path

	// Generate random key
	bs.aes_key = make([]byte, 32)
	_, err := rand.Read(bs.aes_key)

	// Open source file
	fptr, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ProcessFile (path = ", path, " ) (opening file) ", err)
		return err
	}
	// Hash the file
	hasher := sha3.New512()
	if bs.size, err := io.Copy(hasher, fptr); err != nil {
		fmt.Fprintln(os.Stderr, "ProcessFile (path = ", path, " ) (hashing file) ", err)
		return err
	}

	// Store the hash
	bs.hash = hasher.Sum(nil)
	// hxp - HeX encoded Path
	hxp := hex.EncodeToString(hasher.Sum(nil))
	// Make a filename under a few directories to make things faster when reading
	bs.end_path = hxp[0:2] + "/" + hxp[2:4] + "/" + hxp[4:6] + "/" + hxp[6:8] + "/" + hxp

	// Encrypt file

	// Hash encrypted file

	// Store the hash of the encrypted file

	return nil
}

func main() {

}
