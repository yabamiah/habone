package types

import (
	"bytes"
)

type Blockchain struct {
	Blocks []*Block
}

// InitBlockChain creates a new blockchain with genesis block
func InitBlockChain() *Blockchain {
	return &Blockchain{[]*Block{Genesis()}}
}

// Genesis creates the first block in the chain
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

// AddBlock adds a block to the chain
func (chain *Blockchain) AddBlock(data string) (err error) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)

	if !IsBlockValid(*new, *prevBlock) {
		return err
	}
	
	chain.Blocks = append(chain.Blocks, new)
	return nil
}

// Get the block by hash
func (chain *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, block := range chain.Blocks {
		if bytes.Compare(block.Hash, hash) == 0 {
			return block
		}
	}
	return nil
}

// Get the block by index
func (chain *Blockchain) GetBlockByIndex(index int64) *Block {
	for _, block := range chain.Blocks {
		if block.Index == index {
			return block
		}
	}
	return nil
}

// Get the last block in the chain	
func (chain *Blockchain) GetLastBlock() int64 {
	block := chain.Blocks[len(chain.Blocks)-1]
	
	lastBlockIndex:= block.GetIndex()

	return lastBlockIndex
}
