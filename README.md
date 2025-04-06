# Go Merkle Tree

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org/dl/)

A lightweight, efficient Go implementation of Merkle Trees using SHA-256, designed for data integrity verification and efficient dataset comparison.

## Features

- Fast SHA-256 based Merkle Tree implementation
- Proof generation and verification for data integrity
- Efficient comparison of ordered datasets
- Proper handling of odd numbers of leaves
- Zero external dependencies (standard library only)

## Requirements

- Go 1.22 or higher

## Basic Usage

```go
package main

import (
    "fmt"

    merkle "github.com/yourusername/go-merkle-tree"
)

func main() {
    // Create some data
    data := [][]byte{
        []byte("data block 1"),
        []byte("data block 2"),
        []byte("data block 3"),
    }

    // Build the Merkle tree
    tree, err := merkle.NewTree(data)
    if err != nil {
        panic(err)
    }

    // Get the Merkle root
    root := tree.GetRoot()
    fmt.Printf("Merkle Root: %x\n", root)

    // Generate a proof for data block 1
    proof, err := tree.GenerateProof(0)
    if err != nil {
        panic(err)
    }

    // Verify the proof
    valid := merkle.VerifyProof(data[0], proof, root)
    fmt.Printf("Proof verification: %v\n", valid)
}
```

## Advanced Example: Comparing Map Datasets

One powerful application is efficiently comparing two datasets to identify differences:

```go
package main

import (
    "fmt"
    "slices"
    "sort"
    "strings"

    merkle "github.com/yourusername/go-merkle-tree"
)

// Helper to convert map -> ordered [][]byte + sorted keys
func prepareDataBlocks(input map[string]string) ([][]byte, []string, error) {
    // Extract and sort keys for consistent ordering
    keys := make([]string, 0, len(input))
    for k := range input {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Create ordered data blocks
    dataBlocks := make([][]byte, 0, len(keys))
    for _, key := range keys {
        // Serialize as "key:value" pairs
        pair := fmt.Sprintf("%s:%s", key, input[key])
        dataBlocks = append(dataBlocks, []byte(pair))
    }

    return dataBlocks, keys, nil
}

func main() {
    // Two maps with a single difference in value for key "A"
    input1 := map[string]string{"A": "D1", "B": "D2", "C": "D3"}
    input2 := map[string]string{"A": "D7", "B": "D2", "C": "D3"}

    // Prepare ordered data
    dataBlocks1, sortedKeys1, _ := prepareDataBlocks(input1)
    dataBlocks2, sortedKeys2, _ := prepareDataBlocks(input2)

    // Build trees
    tree1, _ := merkle.NewTree(dataBlocks1)
    tree2, _ := merkle.NewTree(dataBlocks2)

    // Compare roots
    root1, root2 := tree1.GetRoot(), tree2.GetRoot()
    if slices.Equal(root1, root2) {
        fmt.Println("Datasets are identical.")
    } else {
        fmt.Println("Datasets differ!")

        // Compare leaves to find specific differences
        leaves1, leaves2 := tree1.GetLeaves(), tree2.GetLeaves()
        if slices.Equal(sortedKeys1, sortedKeys2) { // Ensure keys didn't change
            for i := range leaves1 {
                if !slices.Equal(leaves1[i], leaves2[i]) {
                    // Extract the key from the original data block
                    keyValue := strings.SplitN(string(dataBlocks1[i]), ":", 2)
                    fmt.Printf("  - Difference at key '%s'\n", keyValue[0])
                    fmt.Printf("    Map1: %s\n", input1[keyValue[0]])
                    fmt.Printf("    Map2: %s\n", input2[keyValue[0]])
                }
            }
        } else {
            fmt.Println("Maps have different keys")
            // Find keys present in map1 but not map2
            for _, k := range sortedKeys1 {
                if _, exists := input2[k]; !exists {
                    fmt.Printf("  - Key '%s' exists only in Map1\n", k)
                }
            }
            // Find keys present in map2 but not map1
            for _, k := range sortedKeys2 {
                if _, exists := input1[k]; !exists {
                    fmt.Printf("  - Key '%s' exists only in Map2\n", k)
                }
            }
        }
    }
}
```

## Process Flow

The following diagram illustrates the core flow when comparing datasets:

```
+---------+                     +---------+
|  Map A  |                     |  Map B  |
+---------+                     +---------+
     |                               |
     V [Sort Keys + Serialize Pairs] V [Sort Keys + Serialize Pairs]
+---------------+               +---------------+
| [][]byte A    |               | [][]byte B    |
| (Ordered)     |               | (Ordered)     |
+---------------+               +---------------+
     |                               |
     V [NewTree(dataA)]              V [NewTree(dataB)]
  +-------+                         +-------+
  | TreeA |                         | TreeB |
  +-------+                         +-------+
     |                               |
     V [GetRoot()]                   V [GetRoot()]
  +-------+                         +-------+
  | RootA |                         | RootB |
  +-------+                         +-------+
       \                               /
        \------> [Compare Roots] <----/
                     |
                     V
                +----------+
                | Match?   | === YES ===> [Identical]
                +----------+
                     |
                     NO
                     |
                     V [Compare Leaves(TreeA) vs Leaves(TreeB)]
                     |
                     V
                 [Find Diff @ Index -> Key]
```
