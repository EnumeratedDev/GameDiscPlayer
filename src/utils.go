package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
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

func CopyRecursivelyWithProgress(source, dest string) error {
	paths := make([]string, 0)

	// Discover paths
	err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)

		return nil
	})
	if err != nil {
		return err
	}

	launcher.ProgressWindow.ResetProgressWindow("Copying files...", 0, len(paths))

	for i, path := range paths {

		launcher.ProgressWindow.Set(i)

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

	launcher.ProgressWindow.Set(len(paths))

	return nil
}

func RemoveDirectoryRecursively(dir string) error {
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

	launcher.ProgressWindow.ResetProgressWindow("Removing files...", 0, len(paths))

	for i, path := range paths {

		launcher.ProgressWindow.Set(i)

		fmt.Printf("Removing %s...\n", path)
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	launcher.ProgressWindow.Hide()

	return nil
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

type GithubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GithubAsset `json:"assets"`
}

func GetGithubReleases(url string) (githubReleases []GithubRelease, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)

	err = decoder.Decode(&githubReleases)
	if err != nil {
		return
	}

	return
}
