package types

import (
	"bytes"
	"crypto/sha256"
	"time"
)

// Uint64ToByte converts a uint64 to a byte slice
func Uint64ToByte(val uint64) []byte {
	b := make([]byte, 8)
	for i := uint64(0); i < 8; i++ {
		b[i] = byte((val >> (i * 8)) & 0xff)
	}
	return b
}

func ByteToUint64(b []byte) uint64 {
	var val uint64
	for i := uint64(0); i < 8; i++ {
		val |= uint64(b[i]) << (i * 8)
	}
	return val
}

type Block struct {
	Index     int64
	Hash      []byte // Hash of the block 
	Data      []byte // Data in the block (e.g. transactions)
	PrevHash  []byte // Hash of the previous block (in the chain)
	TimeStamp uint64 // Time the block was created
}

// CreateBlock creates and returns Block
func CreateBlock(data string, prevHash []byte) *Block {
	index := int64(len(prevHash) + 1)
	block := &Block{index, []byte{}, []byte(data), prevHash, uint64(time.Now().Unix()),}
	block.DeriveHash()
	return block
}

// IsBlockValid checks if the block is valid
func IsBlockValid(newB, oldB Block) bool {
	if bytes.Compare(newB.PrevHash, oldB.Hash) != 0 {
		return false
	}
	if bytes.Compare(newB.Hash, newB.DeriveHash()) != 0 {
		return false
	}
	return true
}

// DeriveHash is a function that derives the hash of a block
func (b *Block) DeriveHash() []byte  {
	// Create a byte slice of the block's data
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{},)
	info = bytes.Join([][]byte{info, Uint64ToByte(b.TimeStamp)}, []byte{},)
	hash := sha256.Sum256(info)
	// Set the hash of the block to the hash we just derived
	b.Hash = hash[:]

	return hash[:]
}

// GetIndex returns the index of the block
func (b *Block) GetIndex() int64 {
	if b != nil {
		return b.Index
	}

	return 0
}

func (b *Block) GetTimeStamp() uint64 {
	if b != nil {
		return b.TimeStamp
	}

	return 0
}