package main

import "strings"

type Blob struct {
	Hash    string `json:hash`
	Size    int64  `json:size`
	VolUUID string `json:volume_uuid`
}

func Hash2Path(src_hash string) string {
	parts := strings.Split(src_hash, ":")
	alg := parts[0]
	hash := parts[1]
	return alg + "/" + hash[0:3] + "/" + hash[3:6] + "/" + hash
}
