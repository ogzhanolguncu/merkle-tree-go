package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DirectorySync uses Merkle trees to efficiently sync directories
type DirectorySync struct {
	SourceDir      string
	DestinationDir string
}

// FileInfo stores metadata about a file used for syncing
type FileInfo struct {
	Path         string    // Relative path from root directory
	Size         int64     // File size in bytes
	LastModified time.Time // Last modification time
	IsDir        bool      // Is this a directory
	Hash         []byte    // Hash of file contents (nil for directories)
}

// BuildDirectoryTree scans a directory and builds a list of FileInfo
func (ds *DirectorySync) BuildDirectoryTree(rootDir string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get path relative to root directory
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Normalize path separator for consistency
		relPath = filepath.ToSlash(relPath)

		fileInfo := FileInfo{
			Path:         relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			IsDir:        info.IsDir(),
		}

		// Calculate hash for files, not directories
		if !info.IsDir() {
			hash, err := hashFile(path)
			if err != nil {
				return err
			}
			fileInfo.Hash = hash
		}

		files = append(files, fileInfo)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort files by path for consistent ordering
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// hashFile calculates the SHA-256 hash of a file's contents
func hashFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// BuildMerkleTree creates a Merkle tree from file info list
func (ds *DirectorySync) BuildMerkleTree(files []FileInfo) (*MerkleTree, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to build tree from")
	}

	// Create data blocks from file info
	dataBlocks := make([][]byte, len(files))
	for i, file := range files {
		// For directories, create a special hash based on path + isDir flag
		if file.IsDir {
			h := sha256.New()
			h.Write([]byte(file.Path + ":dir"))
			dataBlocks[i] = h.Sum(nil)
		} else {
			// For files, use the pre-calculated file hash
			dataBlocks[i] = file.Hash
		}
	}

	// Build the Merkle tree
	return NewTree(dataBlocks)
}

// CompareTrees identifies differences between source and destination
func (ds *DirectorySync) CompareTrees(sourceFiles, destFiles []FileInfo) ([]FileInfo, []string, error) {
	// Create maps for quick lookup
	sourceMap := make(map[string]FileInfo)
	destMap := make(map[string]FileInfo)

	for _, file := range sourceFiles {
		sourceMap[file.Path] = file
	}

	for _, file := range destFiles {
		destMap[file.Path] = file
	}

	// Find files to copy (new or modified)
	var filesToCopy []FileInfo
	var filesToDelete []string

	// Find files in source that need to be copied to destination
	for _, file := range sourceFiles {
		destFile, exists := destMap[file.Path]

		// If file doesn't exist in destination or is different, copy it
		if !exists {
			filesToCopy = append(filesToCopy, file)
		} else if !file.IsDir && !bytes.Equal(file.Hash, destFile.Hash) {
			filesToCopy = append(filesToCopy, file)
		}
	}

	// Find files in destination that don't exist in source (to be deleted)
	for _, file := range destFiles {
		_, exists := sourceMap[file.Path]
		if !exists {
			filesToDelete = append(filesToDelete, file.Path)
		}
	}

	return filesToCopy, filesToDelete, nil
}

// SyncDirectories synchronizes files from source to destination
func (ds *DirectorySync) SyncDirectories() error {
	fmt.Println("Building source directory tree...")
	sourceFiles, err := ds.BuildDirectoryTree(ds.SourceDir)
	if err != nil {
		return fmt.Errorf("error scanning source directory: %v", err)
	}

	fmt.Println("Building destination directory tree...")
	destFiles, err := ds.BuildDirectoryTree(ds.DestinationDir)
	if err != nil {
		return fmt.Errorf("error scanning destination directory: %v", err)
	}

	fmt.Println("Building Merkle trees...")
	sourceTree, err := ds.BuildMerkleTree(sourceFiles)
	if err != nil {
		return fmt.Errorf("error building source tree: %v", err)
	}

	destTree, err := ds.BuildMerkleTree(destFiles)
	if err != nil {
		// If destination is empty, just copy everything
		if strings.Contains(err.Error(), "no files") {
			destTree = nil
		} else {
			return fmt.Errorf("error building destination tree: %v", err)
		}
	}

	// Quick check - if root hashes match, directories are identical
	if destTree != nil && bytes.Equal(sourceTree.Root, destTree.Root) {
		fmt.Println("Directories are already in sync.")
		return nil
	}

	fmt.Println("Finding differences...")
	filesToCopy, filesToDelete, err := ds.CompareTrees(sourceFiles, destFiles)
	if err != nil {
		return fmt.Errorf("error comparing trees: %v", err)
	}

	// First create directories
	for _, file := range filesToCopy {
		if file.IsDir {
			destPath := filepath.Join(ds.DestinationDir, file.Path)
			fmt.Printf("Creating directory: %s\n", file.Path)
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %v", destPath, err)
			}
		}
	}

	// Then copy files
	for _, file := range filesToCopy {
		if !file.IsDir {
			srcPath := filepath.Join(ds.SourceDir, file.Path)
			destPath := filepath.Join(ds.DestinationDir, file.Path)

			// Ensure the destination directory exists
			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %v", destDir, err)
			}

			fmt.Printf("Copying file: %s\n", file.Path)
			if err := copyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("error copying %s: %v", file.Path, err)
			}
		}
	}

	// Delete files that don't exist in source
	for _, path := range filesToDelete {
		fullPath := filepath.Join(ds.DestinationDir, path)
		fmt.Printf("Deleting: %s\n", path)
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("error deleting %s: %v", path, err)
		}
	}

	fmt.Println("Sync complete!")
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// Main function to show usage
func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run merkle_sync.go <source_dir> <destination_dir>")
		os.Exit(1)
	}

	sourceDir := os.Args[1]
	destDir := os.Args[2]

	syncer := &DirectorySync{
		SourceDir:      sourceDir,
		DestinationDir: destDir,
	}

	if err := syncer.SyncDirectories(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
