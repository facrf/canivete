package main

import (
	"archive/zip"
	"io"
	"os"
)

func addDirectoryToZip(writer *zip.Writer, directory string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()

	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, err := dir.ReadDir(-1)
	if closeErr := dir.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		input, err := root.Open(entry.Name())
		if err != nil {
			return err
		}
		output, err := writer.Create(entry.Name())
		if err != nil {
			_ = input.Close()
			return err
		}
		if _, err := io.Copy(output, input); err != nil {
			_ = input.Close()
			return err
		}
		if err := input.Close(); err != nil {
			return err
		}
	}

	return nil
}
