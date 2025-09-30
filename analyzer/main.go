package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"sort"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// AccountRLP represents the RLP structure stored in snapshot accounts.
type AccountRLP struct {
	Nonce       uint64
	Balance     []byte
	StorageRoot []byte
	CodeHash    []byte
}

// prefixEnd returns the smallest s such that all keys with `prefix` are < s.
// If prefix is all 0xff, returns nil (no upper bound).
func prefixEnd(prefix []byte) []byte {
	for i := len(prefix) - 1; i >= 0; i-- {
		if prefix[i] != 0xff {
			end := make([]byte, i+1)
			copy(end, prefix[:i+1])
			end[i]++
			return end
		}
	}
	return nil
}

func main() {
	db, err := pebble.Open("./snapshot/chaindata", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Counting storage slots per account...")
	// copy prefix so we don't mutate the package-level constant
	prefix := append([]byte{}, rawdb.SnapshotStoragePrefix...)
	upper := prefixEnd(prefix) // may be nil; pebble accepts nil UpperBound

	iterOpts := &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upper,
	}

	it, err := db.NewIter(iterOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer it.Close()

	counts := map[string]int{}
	for it.First(); it.Valid(); it.Next() {
		k := it.Key()
		// sanity check
		if len(k) < 1+32+32 {
			continue
		}
		acctHash := k[1 : 1+32]                 // slice is valid until Next()
		acctStr := hex.EncodeToString(acctHash) // copied to string immediately
		counts[acctStr]++
		if counts[acctStr] == 1 {
			fmt.Println("New account " + acctStr)
		}
	}
	if err := it.Error(); err != nil {
		log.Fatalf("iterator error: %v", err)
	}

	fmt.Printf("Found %d accounts\n", len(counts))

	fmt.Println("Reading account info and printing top accounts...")
	type kv struct {
		accountHash string
		slots       int
		codeHash    string
	}

	var arr []kv
	for acctHashHex, slotCount := range counts {
		acctHashBytes, _ := hex.DecodeString(acctHashHex)

		// IMPORTANT: create a fresh slice so we don't modify the prefix constant
		accountKey := append([]byte{}, rawdb.SnapshotAccountPrefix...)
		accountKey = append(accountKey, acctHashBytes...)

		// Pebble Get returns (value, closer, error)
		val, closer, err := db.Get(accountKey)
		if err != nil {
			if err == pebble.ErrNotFound {
				// account not found; skip
				closer = nil
			} else {
				log.Printf("db.Get error for %s: %v", acctHashHex, err)
				continue
			}
		}

		var codeHashHex string
		if err == nil {
			// Copy value before closing the closer because Pebble reuses buffers
			accountVal := append([]byte(nil), val...)
			closer.Close()

			var acc AccountRLP
			if err := rlp.DecodeBytes(accountVal, &acc); err == nil {
				codeHashHex = hex.EncodeToString(acc.CodeHash)
			}
		}

		arr = append(arr, kv{
			accountHash: acctHashHex,
			slots:       slotCount,
			codeHash:    codeHashHex,
		})
	}

	// Sort descending by slot count
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].slots > arr[j].slots
	})

	fmt.Println("Top accounts by storage slots:")
	for i, a := range arr {
		if i >= 50 { // top 50
			break
		}
		fmt.Printf("%3d: accountHash=%s slots=%d codeHash=%s\n",
			i+1, a.accountHash, a.slots, a.codeHash)
	}
}
