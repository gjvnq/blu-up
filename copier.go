package main

type CopyOrder struct {
	Origin string
	Dest   string
}

var CopierCh chan CopyOrder

func AddToCopier(origin, hash string) {
	order := CopyOrder{}
	order.Origin = origin
	order.Dest = Hash2Path(hash)
	CopierCh <- order
}
