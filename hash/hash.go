package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: go run hash.go <hexstring>")
	}

	input := os.Args[1]
	input = strings.TrimPrefix(input, "0x") // remove 0x if present

	data, err := hex.DecodeString(input)
	if err != nil {
		log.Fatalf("Invalid hex string: %v", err)
	}

	// Use Geth's crypto.Keccak256
	hash := crypto.Keccak256(data)

	fmt.Printf("0x%s\n", hex.EncodeToString(hash))
}
