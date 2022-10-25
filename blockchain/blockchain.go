package blockchain

import (
	"bytes"
	"crypto/sha256"
	"time"
)

type Blockchain struct {
	Blocks []*Block
}

type Block struct {
	Hash      []byte // Hash of the block 
	Data      []byte // Data in the block (e.g. transactions)
	PrevHash  []byte // Hash of the previous block (in the chain)
	TimeStamp time.Time // Time the block was created
}

// DeriveHash is a function that derives the hash of a block
func (b *Block) DeriveHash() []byte  {
	// Create a byte slice of the block's data
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{},)
	info = bytes.Join([][]byte{info, []byte(b.TimeStamp.String())}, []byte{})
	hash := sha256.Sum256(info)
	// Set the hash of the block to the hash we just derived
	b.Hash = hash[:]

	return hash[:]
}

func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, time.Now()}
	block.DeriveHash()
	return block
}

func (chain *Blockchain) AddBlock(data string) (err error) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)

	if !IsBlockValid(*new, *prevBlock) {
		return err
	}
	chain.Blocks = append(chain.Blocks, new)
	return nil
}

// Genesis creates the first block in the chain
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *Blockchain {
	return &Blockchain{[]*Block{Genesis()}}
}

func IsBlockValid(newB, oldB Block) bool {
	if bytes.Compare(newB.PrevHash, oldB.Hash) != 0 {
		return false
	}
	if bytes.Compare(newB.Hash, newB.DeriveHash()) != 0 {
		return false
	}
	return true
}