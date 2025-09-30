The purpose of this tool is to take Geth's chaindata and to output an analysis of the state there.

Snapshot chaindata can be download from: https://ethpandaops.io/data/snapshots/. Ensure you pick the target network and Geth as client. Place the snapshot after extracting the archive in this folder. The tool reads from `./snapshot/chaindata`.

The goal is to read this snapshot and to provide useful data in the dump of that state snapshot. To name a few:

- The number of accounts in the trie
- The accounts sorted by their storage slot count (size of storage trie). We should measure the evolution over time here for all accounts.
- The amount of unique code hashes on chain
- The most used code hashes
- Bonus: analysis of the code. For instance printing the amount of opcodes and their count. Both when treating each codeHash as unique code or all accounts with the same codeHash as using that opcode multiple times. PUSHx values should be grouped here.

TODO:

Snapshot from EthPandaOps is a geth snapshot with args `--http.api=eth,net,web3,debug --http.vhosts=* --state.scheme=path --cache.preimages`. Ensure that `--state.scheme=path` yields the expected results.

Quick run:

```
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
tar -xzf go1.24.0.linux-amd64.tar.gz
./go/bin/go mod tidy
# Symlink snapshot to here
ln -s /data/client_snapshots/geth/ snapshot
./go/bin/go run ./analyzer/main.go
```

For mainnet on a node with fast SSD on block 23360000, this job started at 01:34:26 and ended at 01:50:33 (commit of this tool: de82e801d14ab8aeacb732411943d82a8b931d70). So it currently takes under 20 minutes to read the mainnet snapshot.