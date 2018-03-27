package main

type Blob struct {
	Hash    string `json:hash`
	VolUUID string `json:volume_uuid`
}
