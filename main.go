package main

import (
	"fmt"
	"github.com/yabamiah/habone/core"
)

func main() {
	chain := core.InitBlockChain()

	if chain.AddBlock("First Block after Genesis") != nil {
		fmt.Println("Error adding block")
	}
	if chain.AddBlock("Second Block after Genesis") != nil {
		fmt.Println("Error adding block")
	}
	if chain.AddBlock("Aprendendo sobre blockchain") != nil {
		fmt.Println("Error adding block")
	}

	for i, block := range chain.Blocks{
		fmt.Println("Block: ", i)
		fmt.Println("Previous Hash:", block.PrevHash)
		fmt.Println("Data in Block:", string(block.Data))
		fmt.Println("Hash:", block.Hash)
	}
}