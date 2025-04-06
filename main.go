package main

import (
	"fmt"
	"log"
	"slices"
)

func main() {
	input := map[string]string{"C": "D3", "B": "D2", "A": "D1"}
	sortedInput := prepareData(input)
	merkle, err := NewTree(sortedInput)
	if err != nil {
		log.Fatal("something went wrong %w", err)
	}
	proofPath, leafHash, _ := merkle.GenerateProof(1)
	log.Printf("proofPath %+v, leafHash %+v \n", proofPath, leafHash)

	result, _ := VerifyProof(merkle.Root, proofPath, leafHash, 1)
	log.Printf("%+v", result)
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
