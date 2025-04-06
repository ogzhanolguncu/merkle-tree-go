// merkle_test.go
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"slices"
	"testing"
)

// Helper to create simple ordered data blocks for testing
func createTestDataBlocks(items ...string) [][]byte {
	blocks := make([][]byte, len(items))
	for i, item := range items {
		// Using simple string conversion for test data blocks
		blocks[i] = []byte(item)
	}
	return blocks
}

func hashData(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func hashPair(h1, h2 []byte) []byte {
	combined := slices.Concat(h1, h2)
	h := sha256.Sum256(combined)
	return h[:]
}

func TestNewTree(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		_, err := NewTree([][]byte{})
		if !errors.Is(err, ErrEmptyMessage) { // Assuming ErrEmptyMessage is the correct error
			t.Errorf("Expected error %v for empty input, got %v", ErrEmptyMessage, err)
		}
	})

	t.Run("SingleLeaf", func(t *testing.T) {
		blocks := createTestDataBlocks("A")
		leafHash := hashData(blocks[0])

		tree, err := NewTree(blocks)
		if err != nil {
			t.Fatalf("NewTree failed for single leaf: %v", err)
		}

		if !bytes.Equal(tree.Root, leafHash) {
			t.Errorf("Expected root %x to equal leaf hash %x", tree.Root, leafHash)
		}
		if len(tree.Leaves) != 1 {
			t.Errorf("Expected 1 leaf, got %d", len(tree.Leaves))
		}
		if !bytes.Equal(tree.Leaves[0], leafHash) {
			t.Errorf("Leaf hash mismatch")
		}
		if !bytes.Equal(tree.nodes[0][0], leafHash) {
			t.Errorf("nodes[0] mismatch")
		}
	})

	t.Run("TwoLeaves", func(t *testing.T) {
		blocks := createTestDataBlocks("A", "B")
		l0 := hashData(blocks[0])
		l1 := hashData(blocks[1])
		expectedRoot := hashPair(l0, l1)

		tree, err := NewTree(blocks)
		if err != nil {
			t.Fatalf("NewTree failed for two leaves: %v", err)
		}

		if !bytes.Equal(tree.Root, expectedRoot) {
			t.Errorf("Root mismatch for two leaves. Expected %x, got %x", expectedRoot, tree.Root)
		}
		if len(tree.Leaves) != 2 {
			t.Errorf("Expected 2 leaves, got %d", len(tree.Leaves))
		}
		if len(tree.nodes) != 2 { // Level 0 (leaves), Level 1 (root)
			t.Errorf("Expected 2 levels in nodes, got %d", len(tree.nodes))
		}
	})

	t.Run("ThreeLeaves", func(t *testing.T) {
		blocks := createTestDataBlocks("A", "B", "C")
		l0 := hashData(blocks[0])
		l1 := hashData(blocks[1])
		l2 := hashData(blocks[2])
		// Calculate expected root manually based on implementation (duplicate last leaf)
		n01 := hashPair(l0, l1)
		n22 := hashPair(l2, l2) // Duplication
		expectedRoot := hashPair(n01, n22)

		tree, err := NewTree(blocks)
		if err != nil {
			t.Fatalf("NewTree failed for three leaves: %v", err)
		}

		if !bytes.Equal(tree.Root, expectedRoot) {
			t.Errorf("Root mismatch for three leaves. Expected %x, got %x", expectedRoot, tree.Root)
		}
		if len(tree.Leaves) != 3 {
			t.Errorf("Expected 3 leaves, got %d", len(tree.Leaves))
		}
		if len(tree.nodes) != 3 { // Level 0 (leaves), Level 1 (parents), Level 2 (root)
			t.Errorf("Expected 3 levels in nodes, got %d", len(tree.nodes))
		}
	})

	t.Run("FourLeaves", func(t *testing.T) {
		blocks := createTestDataBlocks("A", "B", "C", "D")
		l0 := hashData(blocks[0])
		l1 := hashData(blocks[1])
		l2 := hashData(blocks[2])
		l3 := hashData(blocks[3])
		n01 := hashPair(l0, l1)
		n23 := hashPair(l2, l3)
		expectedRoot := hashPair(n01, n23)

		tree, err := NewTree(blocks)
		if err != nil {
			t.Fatalf("NewTree failed for four leaves: %v", err)
		}

		if !bytes.Equal(tree.Root, expectedRoot) {
			t.Errorf("Root mismatch for four leaves. Expected %x, got %x", expectedRoot, tree.Root)
		}
	})
}

func TestGenerateAndVerifyProof(t *testing.T) {
	testCases := []struct {
		name       string
		dataItems  []string
		proveIndex int // Index of leaf to generate/verify proof for
	}{
		{"SingleLeaf", []string{"A"}, 0},
		{"TwoLeaves_Idx0", []string{"A", "B"}, 0},
		{"TwoLeaves_Idx1", []string{"A", "B"}, 1},
		{"ThreeLeaves_Idx0", []string{"A", "B", "C"}, 0},
		{"ThreeLeaves_Idx1", []string{"A", "B", "C"}, 1},
		{"ThreeLeaves_Idx2", []string{"A", "B", "C"}, 2}, // Tests odd duplication path
		{"FourLeaves_Idx0", []string{"A", "B", "C", "D"}, 0},
		{"FourLeaves_Idx1", []string{"A", "B", "C", "D"}, 1},
		{"FourLeaves_Idx2", []string{"A", "B", "C", "D"}, 2},
		{"FourLeaves_Idx3", []string{"A", "B", "C", "D"}, 3},
		{"FiveLeaves_Idx4", []string{"A", "B", "C", "D", "E"}, 4}, // Another odd case
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup ---
			blocks := createTestDataBlocks(tc.dataItems...)
			tree, err := NewTree(blocks)
			if err != nil {
				t.Fatalf("Failed to create tree for test setup: %v", err)
			}
			leafToProve := blocks[tc.proveIndex]
			leafToProveHash := hashData(leafToProve) // VerifyProof needs the leaf HASH

			// --- Test GenerateProof ---
			proofPath, generatedLeafHash, err := tree.GenerateProof(tc.proveIndex)
			if err != nil {
				t.Fatalf("GenerateProof failed: %v", err)
			}
			if !bytes.Equal(generatedLeafHash, leafToProveHash) {
				t.Errorf("GenerateProof returned incorrect leaf hash. Expected %x, got %x", leafToProveHash, generatedLeafHash)
			}
			// Note: Testing exact proof path content requires manual calculation or a trusted reference.
			// Here we primarily test if it verifies correctly. Check length for basic sanity.
			expectedProofLen := 0
			if len(blocks) > 1 {
				// Simple approximation for tree height (log2) - adjust if needed for exactness
				expectedProofLen = len(tree.nodes) - 1
			}
			if len(proofPath) != expectedProofLen {
				t.Errorf("Expected proof path length %d, got %d", expectedProofLen, len(proofPath))
			}

			// --- Test VerifyProof (Valid Case) ---
			isValid, err := VerifyProof(tree.Root, proofPath, leafToProveHash, tc.proveIndex)
			if err != nil {
				t.Errorf("VerifyProof failed for valid proof: %v", err)
			}
			if !isValid {
				t.Errorf("VerifyProof returned false for a valid proof.")
				// Optional: Log details for debugging
				t.Logf("Root: %x", tree.Root)
				t.Logf("Leaf Hash: %x", leafToProveHash)
				t.Logf("Leaf Index: %d", tc.proveIndex)
				t.Logf("Proof Path: %x", proofPath)
			}

			// --- Test VerifyProof (Invalid Cases - only if tree has > 1 node) ---
			if len(blocks) > 1 {
				// Tampered Root
				tamperedRoot := append([]byte{}, tree.Root...)
				tamperedRoot[0] ^= 0xff // Flip first byte
				isTamperedRootValid, errTamperRoot := VerifyProof(tamperedRoot, proofPath, leafToProveHash, tc.proveIndex)
				if errTamperRoot != nil {
					t.Errorf("VerifyProof (TamperedRoot) returned error: %v", errTamperRoot)
				}
				if isTamperedRootValid {
					t.Errorf("VerifyProof (TamperedRoot) returned true for tampered root.")
				}

				// Tampered Leaf Hash
				tamperedLeafHash := append([]byte{}, leafToProveHash...)
				tamperedLeafHash[0] ^= 0xff
				isTamperedLeafValid, errTamperLeaf := VerifyProof(tree.Root, proofPath, tamperedLeafHash, tc.proveIndex)
				if errTamperLeaf != nil {
					t.Errorf("VerifyProof (TamperedLeaf) returned error: %v", errTamperLeaf)
				}
				if isTamperedLeafValid {
					t.Errorf("VerifyProof (TamperedLeaf) returned true for tampered leaf hash.")
				}

				// Tampered Proof Path (modify first sibling hash)
				if len(proofPath) > 0 {
					tamperedProofPath := make([][]byte, len(proofPath))
					copy(tamperedProofPath, proofPath)
					tamperedSibling := append([]byte{}, proofPath[0]...)
					tamperedSibling[0] ^= 0xff
					tamperedProofPath[0] = tamperedSibling
					isTamperedPathValid, errTamperPath := VerifyProof(tree.Root, tamperedProofPath, leafToProveHash, tc.proveIndex)
					if errTamperPath != nil {
						t.Errorf("VerifyProof (TamperedPath) returned error: %v", errTamperPath)
					}
					if isTamperedPathValid {
						t.Errorf("VerifyProof (TamperedPath) returned true for tampered proof path.")
					}

					// Empty sibling in path
					proofWithEmptySibling := make([][]byte, len(proofPath))
					copy(proofWithEmptySibling, proofPath)
					proofWithEmptySibling[0] = []byte{} // Make first sibling empty
					isEmptySiblingValid, errEmptySibling := VerifyProof(tree.Root, proofWithEmptySibling, leafToProveHash, tc.proveIndex)
					if errEmptySibling == nil || !errors.Is(errEmptySibling, ErrInvalidProof) {
						t.Errorf("VerifyProof (EmptySibling) expected ErrInvalidProof, got %v", errEmptySibling)
					}
					if isEmptySiblingValid {
						t.Errorf("VerifyProof (EmptySibling) returned true.")
					}
				}

				// Incorrect Leaf Index
				wrongIndex := (tc.proveIndex + 1) % len(blocks) // Just pick another index
				isWrongIndexValid, errWrongIndex := VerifyProof(tree.Root, proofPath, leafToProveHash, wrongIndex)
				if errWrongIndex != nil {
					t.Errorf("VerifyProof (WrongIndex) returned error: %v", errWrongIndex)
				}
				if isWrongIndexValid {
					t.Errorf("VerifyProof (WrongIndex) returned true for wrong leaf index.")
				}
			}
		})
	}
}

func TestGenerateProofEdgeCases(t *testing.T) {
	blocks := createTestDataBlocks("A", "B", "C")
	tree, err := NewTree(blocks)
	if err != nil {
		t.Fatalf("Test setup failed: %v", err)
	}

	t.Run("IndexNegative", func(t *testing.T) {
		_, _, err := tree.GenerateProof(-1)
		if !errors.Is(err, ErrOutOfBoundary) {
			t.Errorf("Expected ErrOutOfBoundary for negative index, got %v", err)
		}
	})

	t.Run("IndexTooLarge", func(t *testing.T) {
		_, _, err := tree.GenerateProof(len(tree.Leaves)) // Index == len is out of bounds
		if !errors.Is(err, ErrOutOfBoundary) {
			t.Errorf("Expected ErrOutOfBoundary for index >= len, got %v", err)
		}
	})
}

func TestVerifyProofEdgeCases(t *testing.T) {
	// Single leaf tree setup
	blocks1 := createTestDataBlocks("A")
	tree1, _ := NewTree(blocks1)
	leafHash1 := hashData(blocks1[0])

	// Two leaf tree setup (for non-empty proof test)
	blocks2 := createTestDataBlocks("A", "B")
	tree2, _ := NewTree(blocks2)
	leafHash2_0 := hashData(blocks2[0])
	proof2_0, _, _ := tree2.GenerateProof(0) // Valid proof for testing invalid inputs

	t.Run("InvalidInput_EmptyRoot", func(t *testing.T) {
		_, err := VerifyProof(nil, [][]byte{}, leafHash1, 0) // Using single leaf case
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty root, got %v", err)
		}
		_, err = VerifyProof([]byte{}, [][]byte{}, leafHash1, 0)
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty root slice, got %v", err)
		}
	})

	t.Run("InvalidInput_EmptyLeaf", func(t *testing.T) {
		_, err := VerifyProof(tree1.Root, [][]byte{}, nil, 0)
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty leaf hash, got %v", err)
		}
		_, err = VerifyProof(tree1.Root, [][]byte{}, []byte{}, 0)
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty leaf hash slice, got %v", err)
		}
	})

	t.Run("ValidEmptyProof", func(t *testing.T) {
		// Single leaf tree: proof path is empty, leaf hash IS the root
		isValid, err := VerifyProof(tree1.Root, [][]byte{}, leafHash1, 0)
		if err != nil {
			t.Errorf("VerifyProof failed for valid empty proof: %v", err)
		}
		if !isValid {
			t.Errorf("VerifyProof returned false for valid empty proof.")
		}
	})

	t.Run("InvalidEmptyProof", func(t *testing.T) {
		// Single leaf tree, but provide wrong leaf hash
		wrongLeafHash := hashData([]byte("WrongData"))
		isValid, err := VerifyProof(tree1.Root, [][]byte{}, wrongLeafHash, 0)
		if err != nil {
			t.Errorf("VerifyProof failed for invalid empty proof: %v", err)
		}
		if isValid {
			t.Errorf("VerifyProof returned true for invalid empty proof.")
		}
	})

	t.Run("NonEmptyProofButEmptyInputs", func(t *testing.T) {
		// Test error precedence: Ensure empty root/leaf error happens before proof path check if path isn't empty
		_, err := VerifyProof(nil, proof2_0, leafHash2_0, 0)
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty root with non-empty proof, got %v", err)
		}
		_, err = VerifyProof(tree2.Root, proof2_0, nil, 0)
		if !errors.Is(err, ErrInvalidProofInputs) {
			t.Errorf("Expected ErrInvalidProofInputs for empty leaf with non-empty proof, got %v", err)
		}
	})
}
