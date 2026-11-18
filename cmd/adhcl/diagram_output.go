package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func deliverDiagram(out io.Writer, path string, page []byte) error {
	if path == "" {
		n, err := out.Write(page)
		if err == nil && n != len(page) {
			return io.ErrShortWrite
		}
		return err
	}
	mode := os.FileMode(0644)
	info, err := os.Stat(path)
	if err == nil {
		mode = info.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	} else if !os.IsNotExist(err) {
		return diagramFileError(path, err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".adhcl-diagram-*")
	if err != nil {
		return diagramFileError(path, err)
	}
	defer os.Remove(f.Name())
	n, err := f.Write(page)
	if err == nil && n != len(page) {
		err = io.ErrShortWrite
	}
	if err != nil {
		_ = f.Close()
		return diagramFileError(path, err)
	}
	if err = f.Chmod(mode); err != nil {
		_ = f.Close()
		return diagramFileError(path, err)
	}
	if err = f.Close(); err != nil {
		return diagramFileError(path, err)
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return diagramFileError(path, err)
	}
	return nil
}

func diagramFileError(path string, err error) error {
	switch e := err.(type) {
	case *os.PathError:
		err = e.Err
	case *os.LinkError:
		err = e.Err
	}
	return fmt.Errorf("%s: %w", path, err)
}
