package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"time"

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

// progressFraction returns a fraction [0,1] representing the position of current
// lexicographically between lower and upper using arbitrary-precision arithmetic.
func progressFraction(current, lower, upper []byte) float64 {
	// determine maximum length
	n := len(current)
	if len(lower) > n {
		n = len(lower)
	}
	if len(upper) > n {
		n = len(upper)
	}

	// right-pad slices to length n
	c := make([]byte, n)
	copy(c, current) // original bytes at the left, zeros appended at the right
	l := make([]byte, n)
	copy(l, lower)
	u := make([]byte, n)
	copy(u, upper)

	// convert to big.Int (big-endian)
	num := new(big.Int).SetBytes(c)
	lowerInt := new(big.Int).SetBytes(l)
	upperInt := new(big.Int).SetBytes(u)

	// compute fraction
	num.Sub(num, lowerInt)                        // num = current - lower
	denom := new(big.Int).Sub(upperInt, lowerInt) // denom = upper - lower

	if denom.Sign() <= 0 {
		return 1.0
	}

	// convert to float
	fraction, _ := new(big.Rat).SetFrac(num, denom).Float64()

	// clamp to [0,1]
	if fraction < 0 {
		return 0
	}
	if fraction > 1 {
		return 1
	}
	return fraction
}

func main() {
	db, err := pebble.Open("./snapshot/chaindata", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Counting storage slots per account...")
	prefix := append([]byte{}, rawdb.SnapshotStoragePrefix...)
	upper := prefixEnd(prefix)

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
	lastUpdate := time.Now()
	updateInterval := 500 * time.Millisecond // throttle updates
	var lastAcct string

	for it.First(); it.Valid(); it.Next() {
		k := it.Key()
		if len(k) < 1+32+32 {
			continue
		}
		acctHash := k[1 : 1+32]
		acctStr := hex.EncodeToString(acctHash)
		counts[acctStr]++

		// periodically update progress
		if time.Since(lastUpdate) > updateInterval {
			// Account-level progress (approximate)
			accFrac := progressFraction(k, prefix, upper)
			processedAccounts := len(counts) // how many unique accounts see
			if acctStr == lastAcct {
				// Storage-level progress for the current account
				storageLower := append([]byte{prefix[0]}, acctHash...)
				storageUpper := make([]byte, 1+32+32)
				copy(storageUpper, storageLower)
				for i := 33; i < 1+32+32; i++ {
					storageUpper[i] = 0xff
				}
				storageFrac := progressFraction(k, storageLower, storageUpper)
				// Print accounts processed + storage progress
				fmt.Printf("\rAccounts %d processed [%6.2f%%] (This account storage: [%6.2f%%]) ",
					processedAccounts, accFrac*100, storageFrac*100)
			} else {
				// Account-level progress only
				fmt.Printf("\rAccounts %d processed [%6.2f%%]                                   ",
					processedAccounts, accFrac*100)
				lastAcct = acctStr
			}
			lastUpdate = time.Now()
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
