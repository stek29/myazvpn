package nftmap

type NFTMapElem struct {
	Key, Val uint32
}

type NFTMapper interface {
	Clear() error
	List() ([]NFTMapElem, error)
	Has(k uint32) (bool, error)
	Add(elems ...NFTMapElem) error
	Remove(k uint32) error
	Close() error
}

// NewNFTMapper opens an nftables conn
// ensures given table exists
// and ensures it has a map with given name
func NewNFTMapper(table, name string) (NFTMapper, error) {
	return newNFTMapper(table, name)
}
