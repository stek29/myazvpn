package nftmap

import (
	"fmt"
	"net"

	"github.com/google/nftables"
)

func elemFromNftables(elems []nftables.SetElement) []NFTMapElem {
	res := make([]NFTMapElem, len(elems))
	for i, elem := range elems {
		res[i].Key = ip4ToUint32(net.IP(elem.Key))
		res[i].Val = ip4ToUint32(net.IP(elem.Val))
	}
	return res
}

func elemToNftables(elems []NFTMapElem) []nftables.SetElement {
	res := make([]nftables.SetElement, len(elems))
	for i, elem := range elems {
		res[i].Key = uint32ToIP4(elem.Key)
		res[i].Val = uint32ToIP4(elem.Val)
	}
	return res
}

type nftMapper struct {
	c *nftables.Conn

	table *nftables.Table
	set   *nftables.Set
}

func newNFTMapper(table, name string) (NFTMapper, error) {
	var m nftMapper

	m.c = &nftables.Conn{}

	m.table = m.c.AddTable(&nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   table,
	})

	m.set = &nftables.Set{
		Table:    m.table,
		Name:     name,
		IsMap:    true,
		KeyType:  nftables.TypeIPAddr,
		DataType: nftables.TypeIPAddr,
	}

	err := m.c.AddSet(m.set, nil)
	if err != nil {
		return nil, fmt.Errorf("conn.AddSet: %w", err)
	}

	err = m.c.Flush()
	if err != nil {
		return nil, fmt.Errorf("conn.Flush after conn.AddSet: %w", err)
	}

	return &m, nil
}

func (m *nftMapper) Clear() error {
	m.c.FlushSet(m.set)
	err := m.c.Flush()
	if err != nil {
		return fmt.Errorf("conn.Flush after conn.FlushSet: %w", err)
	}
	return nil
}

func (m *nftMapper) List() ([]NFTMapElem, error) {
	elems, err := m.c.GetSetElements(m.set)
	if err != nil {
		return nil, fmt.Errorf("conn.GetSetElements: %w", err)
	}
	return elemFromNftables(elems), nil
}

func (m *nftMapper) Has(k uint32) (bool, error) {
	return false, fmt.Errorf("Has is not impelemented")
}

func (m *nftMapper) Add(elems ...NFTMapElem) error {
	nftElems := elemToNftables(elems)
	err := m.c.SetAddElements(m.set, nftElems)
	if err != nil {
		return fmt.Errorf("conn.SetAddElements: %w", err)
	}
	return nil
}

func (m *nftMapper) Remove(k uint32) error {
	return fmt.Errorf("Remove is not impelemented")
}

func (m *nftMapper) Close() error {
	err := m.c.Flush()
	if err != nil {
		return fmt.Errorf("conn.Flush: %w", err)
	}

	err = m.c.CloseLasting()
	if err != nil {
		return fmt.Errorf("conn.CloseLasting: %w", err)
	}

	return nil
}
