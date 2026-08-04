package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

func Copy(source, dest string) error {
	stat, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !stat.Mode().IsRegular() {
		return fmt.Errorf("unknown file type")
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	destFile.Chmod(stat.Mode().Perm())

	return nil
}

func CopyRecursivelyWithProgress(source, dest string, progressWindow ProgressWindow) error {
	paths := make([]string, 0)

	// Discover paths
	err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)

		return nil
	})
	if err != nil {
		return err
	}

	progressWindow.SetTotal(len(paths))
	progressWindow.SetStatus("Copying files...")

	for i, path := range paths {

		progressWindow.Set(i)

		stat, err := os.Lstat(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		if stat.Mode().IsDir() {
			fmt.Printf("Creating directory at %s...\n", filepath.Join(dest, relPath))
			err = os.Mkdir(filepath.Join(dest, relPath), stat.Mode().Perm())
			if err != nil {
				return err
			}
		} else if stat.Mode().IsRegular() {
			fmt.Printf("Copying %s to %s...\n", path, filepath.Join(dest, relPath))
			err = Copy(path, filepath.Join(dest, relPath))
			if err != nil {
				return err
			}
		} else if stat.Mode().Type()&os.ModeSymlink != 0 {
			symlinkPath, err := os.Readlink(path)
			if err != nil {
				return err
			}
			fmt.Printf("Symlinking %s to %s...\n", filepath.Join(dest, relPath), symlinkPath)
			os.Symlink(symlinkPath, filepath.Join(dest, relPath))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown file type")
		}
	}

	progressWindow.Set(len(paths))

	return nil
}

func RemoveDirectoryRecursively(dir string, progressWindow ProgressWindow) error {
	paths := make([]string, 0)

	// Discover paths
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)

		return nil
	})
	if err != nil {
		return err
	}

	slices.Reverse(paths)

	progressWindow.SetTotal(len(paths))
	progressWindow.SetStatus("Removing files...")

	for i, path := range paths {

		progressWindow.Set(i)

		fmt.Printf("Removing %s...\n", path)
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	progressWindow.Set(len(paths))

	return nil
}
