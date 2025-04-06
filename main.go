package main

import (
	"fmt"
	"slices"
	"sort"
	// For simple serialization example
	// Include your MerkleTree struct and functions here...
)

// Helper function to prepare data blocks from a map
func prepareDataBlocks(input map[string]string) ([][]byte, []string, error) {
	if len(input) == 0 {
		// Decide how to handle empty maps - maybe return ErrEmptyMessage?
		// Or return empty slices if that's valid for your comparison logic.
		return [][]byte{}, []string{}, nil // Or return an error if empty maps aren't comparable
	}

	// 1. Get keys
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}

	// 2. Sort keys consistently
	sort.Strings(keys)

	// 3. Create ordered data blocks
	dataBlocks := make([][]byte, 0, len(keys))
	for _, k := range keys {
		v := input[k]
		// 4. Serialize consistently (e.g., key:value)
		//    IMPORTANT: Choose a robust serialization for real-world use.
		//    This simple example assumes ':' doesn't appear in keys/values.
		serializedPair := fmt.Appendf(nil, "%s:%s", k, v)
		dataBlocks = append(dataBlocks, serializedPair)
	}

	// Return both the blocks and the order of keys used
	return dataBlocks, keys, nil
}

func main() {
	// Your two datasets
	input1 := map[string]string{"C": "D3", "B": "D2", "A": "D1"}
	input2 := map[string]string{"C": "D3", "B": "D2", "A": "D7"} // Difference in A

	// Prepare data for Merkle Tree construction
	dataBlocks1, sortedKeys1, err1 := prepareDataBlocks(input1)
	if err1 != nil {
		fmt.Println("Error preparing data 1:", err1)
		return
	}

	dataBlocks2, sortedKeys2, err2 := prepareDataBlocks(input2)
	if err2 != nil {
		fmt.Println("Error preparing data 2:", err2)
		return
	}

	// --- Build Trees using your implementation ---
	tree1, errTree1 := NewTree(dataBlocks1)
	if errTree1 != nil {
		fmt.Println("Error building tree 1:", errTree1)
		return
	}
	root1 := tree1.GetRoot()
	fmt.Printf("Tree 1 Root: %x\n", root1)

	tree2, errTree2 := NewTree(dataBlocks2)
	if errTree2 != nil {
		fmt.Println("Error building tree 2:", errTree2)
		return
	}
	root2 := tree2.GetRoot()
	fmt.Printf("Tree 2 Root: %x\n", root2)

	// --- Compare Roots ---
	if slices.Equal(root1, root2) {
		fmt.Println("\nDatasets are identical.")
	} else {
		fmt.Println("\nDatasets differ!")

		// --- Find the Faulty Section using current implementation ---
		// We compare the leaves. Since the data was ordered by sorted keys,
		// the leaf indices correspond to the sorted key indices.
		leaves1 := tree1.GetLeaves()
		leaves2 := tree2.GetLeaves()

		if len(leaves1) != len(leaves2) {
			fmt.Printf("Difference in number of items: %d vs %d\n", len(leaves1), len(leaves2))
			// More complex diffing might be needed if keys themselves differ
		} else if !slices.Equal(sortedKeys1, sortedKeys2) {
			fmt.Println("Difference in the set of keys!")
			// Add logic here to show which keys are added/removed
		} else {
			fmt.Println("Differences found in values:")
			for i := range leaves1 {
				if !slices.Equal(leaves1[i], leaves2[i]) {
					// Found a difference! Map index back to the key.
					differingKey := sortedKeys1[i] // or sortedKeys2[i], they are the same
					// Note: leaves are HASHES of the serialized data.
					// To show the actual differing values, you might need the original dataBlocks
					originalValue1 := string(dataBlocks1[i]) // Shows "key:value"
					originalValue2 := string(dataBlocks2[i]) // Shows "key:value"

					fmt.Printf("  - Difference at key '%s':\n", differingKey)
					fmt.Printf("    Dataset 1 (%x): %s\n", leaves1[i], originalValue1)
					fmt.Printf("    Dataset 2 (%x): %s\n", leaves2[i], originalValue2)
				}
			}
		}
	}
}
