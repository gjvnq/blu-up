package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

	"golang.org/x/crypto/sha3"
)

type BinSeq struct {
	raw_hash     []byte // SHA3-256 of the original file
	raw_size     int64  // Size in bytes of the original file
	is_encrypted bool
	aes_key      []byte
	hash         []byte // SHA3-256 of the encripted file (or original if not encrypted)
	size         int64  // Size in bytes of the encripted file (or original if not encrypted)
	path         string
}

type EncBinSeq struct {
}

func (bs BinSeq) GetKey() string {
	return hex.EncodeToString(bs.aes_key)
}

func (bs *BinSeq) ProcessFromFile(input_path, output_dir, encrypt string) error {
	bs.is_encrypted = false

	// Open source file
	in_fptr, err := os.Open(input_path)
	defer in_fptr.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (opening file): %s ", input_path, output_dir, err)
		return err
	}
	// Hash the file
	hasher := sha3.New256()
	if bs.size, err = io.Copy(hasher, in_fptr); err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (hashing file): %s ", input_path, output_dir, err)
		return err
	}

	// Store the hash
	bs.raw_hash = hasher.Sum(nil)
	// hxp - HeX encoded Path
	hxp := hex.EncodeToString(bs.raw_hash)
	// Make a filename under a few directories to make things faster when reading
	bs.path = hxp[0:2] + "/" + hxp[2:4] + "/" + hxp[4:6] + "/" + hxp[6:8] + "/" + hxp

	// Create temporary file for storing the encrypted file
	out_fptr, err := ioutil.TempFile(output_dir, "")
	defer out_fptr.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (creating temporary file): %s ", input_path, output_dir, err)
		return err
	}

	// Setup things for encription
	bs.aes_key = bs.raw_hash
	bs.is_encrypted = true
	block, err := aes.NewCipher(bs.aes_key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (aes.NewCipher): %s ", input_path, output_dir, err)
		return err
	}
	iv := make([]byte, aes.BlockSize)
	_, err = rand.Read(bs.aes_key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (creating iv): %s ", input_path, output_dir, err)
		return err
	}
	stream := cipher.NewOFB(block, iv[:])
	writer := &cipher.StreamWriter{S: stream, W: out_fptr}
	// Now, encrypt
	if _, err := io.Copy(writer, in_fptr); err != nil {
		fmt.Fprintf(os.Stderr, "ProcessFromFile(input_path = '%s', output_dir = '%s' ) (io.Copy): %s ", input_path, output_dir, err)
		return err
	}

	// Hash encrypted file

	// Store the hash of the encrypted file

	return nil
}
