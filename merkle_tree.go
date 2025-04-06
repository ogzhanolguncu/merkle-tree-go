package main

import (
	"crypto/sha256"
	"errors"
	"slices"
)

// MerkleTree holds the computed hashes and structure of a Merkle Tree.
type MerkleTree struct {
	// Root: The final root hash of the tree ([]byte).
	// This is the primary identifier for the tree's state.
	Root []byte

	// Leaves: An ordered slice containing the initial leaf hashes ([]byte).
	// This represents the hashed data blocks at the bottom level (Level 0).
	Leaves [][]byte

	// nodes: Stores all computed hashes, organized by level ([][][]byte).
	//        - nodes[0] is the leaf level (should be identical to Leaves).
	//        - nodes[1] contains the hashes from the level above the leaves, etc.
	//        - nodes[len(nodes)-1] contains a single element: the Root.
	// Storing all nodes is necessary for efficient proof generation.
	nodes [][][]byte
}

var (
	ErrEmptyMessage       = errors.New("merkleTree: empty dataBlocks")
	ErrInsufficientLevel  = errors.New("merkleTree: input level must have more than one hash")
	ErrZeroLeaves         = errors.New("merkleTree: cannot calculate tree with zero leaves")
	ErrOutOfBoundary      = errors.New("merkleTree: leaf is out of boundary")
	ErrHashOrProof        = errors.New("merkleTree: empty hash or proof")
	ErrInvalidProofInputs = errors.New("merkleTree: invalid inputs: expected root, leaf hash cannot be empty")
	ErrInvalidProof       = errors.New("merkleTree: invalid proof: contains empty sibling hash")
	ErrProofPathRequired  = errors.New("merkleTree: proof path cannot be nil (use empty slice for single-node tree)") // Example if nil proofPath is invalid
)

// NewTree creates a new Merkle Tree from ordered data blocks.
// It assumes dataBlocks are already serialized and deterministically ordered
// by the caller (e.g., based on sorted keys or file paths).
// It calculates all necessary hashes and populates the MerkleTree struct.
func NewTree(dataBlocks [][]byte) (*MerkleTree, error) {
	merkle := &MerkleTree{}

	if len(dataBlocks) == 0 {
		return nil, ErrEmptyMessage
	}
	merkle.Leaves = hashLeaves(dataBlocks)
	nodes, err := calculateTreeLevels(merkle.Leaves)
	if err != nil {
		return nil, err
	}

	merkle.nodes = nodes
	merkle.Root = nodes[len(nodes)-1][0]

	return merkle, nil
}

// GetRoot returns the root hash of the tree.
func (t *MerkleTree) GetRoot() []byte {
	if t.Root == nil {
		return nil
	}
	root := make([]byte, len(t.Root))
	copy(root, t.Root)
	return root
}

// GetLeaves returns the ordered slice of leaf hashes.
func (t *MerkleTree) GetLeaves() [][]byte {
	leaves := make([][]byte, 0, len(t.Leaves))
	for _, leaf := range t.Leaves {
		leafCopy := make([]byte, len(leaf))
		copy(leafCopy, leaf)
		leaves = append(leaves, leafCopy)
	}
	return leaves
}

// GenerateProof creates the authentication path (Merkle proof) for the leaf
// at the specified index. The proof consists of the sibling hashes required
// to hash up to the root. The path is ordered from bottom (leaf sibling) to top.
func (t *MerkleTree) GenerateProof(leafIndex int) (proofPath [][]byte, leafHash []byte, err error) {
	if leafIndex >= len(t.Leaves) || leafIndex < 0 {
		return nil, nil, ErrOutOfBoundary
	}

	leafHash = t.Leaves[leafIndex]
	proofPath = make([][]byte, 0)
	currentIndex := leafIndex

	for level := range len(t.nodes) - 1 {
		currentLevelNodes := t.nodes[level]
		var siblingIndex int
		if currentIndex%2 == 0 {
			siblingIndex = currentIndex + 1 // Sibling is to the right
		} else {
			siblingIndex = currentIndex - 1 // Sibling is to the left
		}

		var siblingHash []byte
		if siblingIndex < 0 || siblingIndex >= len(currentLevelNodes) {
			// This happens when currentIndex was the last node on an odd-sized level below.
			// The node itself was paired with a duplicate of itself.
			// So, the hash needed for the proof path is the node's own hash.
			siblingHash = currentLevelNodes[currentIndex]
		} else {
			// Normal case: sibling exists within bounds.
			siblingHash = currentLevelNodes[siblingIndex]
		}
		proofPath = append(proofPath, siblingHash)
		currentIndex = currentIndex / 2
	}

	return proofPath, leafHash, nil
}

// VerifyProof checks if a given leaf hash and its corresponding proof path
// correctly hash up to the expected root hash.
// `expectedRoot`: The trusted root hash of the Merkle Tree.
// `proofPath`: The slice of sibling hashes (ordered bottom-up) from GenerateProof.
// `leafHash`: The hash of the data block being verified.
// `leafIndex`: The original index of the leaf within the tree's ordered leaves.
//
//	This index is crucial for determining hash concatenation order (left vs right).
func VerifyProof(expectedRoot []byte, proofPath [][]byte, leafHash []byte, leafIndex int) (bool, error) {
	if len(expectedRoot) == 0 || len(leafHash) == 0 {
		return false, ErrInvalidProofInputs
	}
	if len(proofPath) == 0 {
		isValid := slices.Equal(leafHash, expectedRoot)
		return isValid, nil
	}

	currentHash := leafHash
	currentIndex := leafIndex

	for _, siblingHash := range proofPath {
		if len(siblingHash) == 0 { // Good to also check inside loop
			return false, ErrInvalidProof
		}
		isRightNode := currentIndex%2 != 0

		var concatted []byte
		if isRightNode {
			concatted = slices.Concat(siblingHash, currentHash)
		} else {
			concatted = slices.Concat(currentHash, siblingHash)
		}
		computedHash := sha256.Sum256(concatted)

		currentHash = computedHash[:]
		currentIndex = currentIndex / 2
	}

	return slices.Equal(currentHash, expectedRoot), nil // Placeholder
}

// hashLeaves calculates the SHA256 hash for each data block.
func hashLeaves(dataBlocks [][]byte) [][]byte {
	leaves := make([][]byte, 0, len(dataBlocks))
	for _, input := range dataBlocks {
		hash := sha256.Sum256(input)
		leaves = append(leaves, hash[:])
	}
	return leaves
}

// calculateTreeLevels builds all levels of the Merkle tree from the leaf hashes.
func calculateTreeLevels(leaves [][]byte) ([][][]byte, error) {
	if len(leaves) == 0 {
		return nil, ErrZeroLeaves
	}
	allLevels := make([][][]byte, 0)
	allLevels = append(allLevels, leaves)

	currentLevel := leaves
	for len(currentLevel) > 1 {
		nextLevel, err := calculateNextLevel(currentLevel)
		if err != nil {
			return nil, err
		}
		allLevels = append(allLevels, nextLevel)
		currentLevel = nextLevel
	}
	return allLevels, nil
}

// calculateNextLevel computes the next level hashes from the current level.
func calculateNextLevel(currentLevelHashes [][]byte) ([][]byte, error) {
	if len(currentLevelHashes) <= 1 {
		return nil, ErrInsufficientLevel
	}

	levelToProcess := currentLevelHashes
	if len(currentLevelHashes)%2 != 0 {
		levelToProcess = make([][]byte, len(currentLevelHashes), len(currentLevelHashes)+1)
		copy(levelToProcess, currentLevelHashes)
		levelToProcess = append(levelToProcess, currentLevelHashes[len(currentLevelHashes)-1])
	}

	nextLevelHashes := make([][]byte, 0, len(levelToProcess)/2)

	for i := 0; i < len(levelToProcess); i += 2 {
		hash1 := levelToProcess[i]
		hash2 := levelToProcess[i+1]

		concattedPair := slices.Concat(hash1, hash2)

		newHash := sha256.Sum256(concattedPair)
		nextLevelHashes = append(nextLevelHashes, newHash[:])
	}

	return nextLevelHashes, nil
}
