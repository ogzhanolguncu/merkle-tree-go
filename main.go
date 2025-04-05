package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"slices"
)

func main() {
	input := map[string]string{"C": "D3", "B": "D2", "A": "D1"}
	sortedInput := prepareData(input)
	leafs := generateLeafs(sortedInput)
	rootHash := calculateRootHash(leafs)
	log.Printf("rootHash %+v", rootHash)
}

// This function sorts keys deterministically, creates and ordered slice of serialized data blocks
func prepareData(input map[string]string) [][]byte {
	// Get keys
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}

	// Sort keys
	slices.Sort(keys)

	serializedInput := make([][]byte, 0, len(input))
	// Iterate sorted keys and serialize
	for _, key := range keys {
		value := input[key]
		serializedInput = append(serializedInput, fmt.Appendf(nil, "%s%s", key, value))
	}

	return serializedInput
}

func generateLeafs(sortedInput [][]byte) [][]byte {
	leafs := make([][]byte, 0, len(sortedInput))
	for _, input := range sortedInput {
		hash := sha256.Sum256(input)
		leafs = append(leafs, hash[:])
	}
	return leafs
}

func calculateRootHash(levelHashes [][]byte) []byte {
	// If only one hash remains, it's the root
	if len(levelHashes) == 1 {
		return levelHashes[0]
	}

	// Make the tree balanced
	if len(levelHashes)%2 != 0 {
		levelHashes = append(levelHashes, levelHashes[len(levelHashes)-1])
	}

	nextLevelHashes := make([][]byte, 0, len(levelHashes)/2)

	for i := 0; i < len(levelHashes); i += 2 {
		hash1 := levelHashes[i]
		hash2 := levelHashes[i+1]

		concattedPair := slices.Concat(hash1, hash2)

		newHash := sha256.Sum256(concattedPair)
		nextLevelHashes = append(nextLevelHashes, newHash[:])
	}

	return calculateRootHash(nextLevelHashes)
}
