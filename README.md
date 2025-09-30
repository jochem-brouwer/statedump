The purpose of this tool is to take Geth's chaindata and to output an analysis of the state there.

Snapshot chaindata can be download from: https://ethpandaops.io/data/snapshots/. Ensure you pick the target network and Geth as client. Place the snapshot after extracting the archive in this folder. The tool reads from `./snapshot/chaindata`.

The goal is to read this snapshot and to provide useful data in the dump of that state snapshot. To name a few:

- The number of accounts in the trie
- The accounts sorted by their storage slot count (size of storage trie). We should measure the evolution over time here for all accounts.
- The amount of unique code hashes on chain
- The most used code hashes
- Bonus: analysis of the code. For instance printing the amount of opcodes and their count. Both when treating each codeHash as unique code or all accounts with the same codeHash as using that opcode multiple times. PUSHx values should be grouped here.