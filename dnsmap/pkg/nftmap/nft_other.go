//go:build !linux

package nftmap

import "fmt"

func newNFTMapper(table, name string) (NFTMapper, error) {
	return nil, fmt.Errorf("NFTMapper is only supported on linux")
}
