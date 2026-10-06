package media

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// cleanedMarker records that the photos already in a directory were
// cleaned, so restarts don't re-encode them again (each pass costs quality).
const cleanedMarker = ".metadata-cleaned"

// CleanExisting removes metadata from every photo already stored under dir,
// in place: names stay the same, so stored links keep working. It runs once
// per directory. Files that aren't images are left alone and reported.
func CleanExisting(dir string) (int, error) {
	marker := filepath.Join(dir, cleanedMarker)
	if _, err := os.Stat(marker); err == nil {
		return 0, nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0, nil
	}

	cleaned := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == cleanedMarker {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		out := &bytes.Buffer{}
		_, cleanErr := Clean(f, out)
		f.Close()
		if cleanErr != nil {
			log.Printf("photo %s left as is: %v", path, cleanErr)
			return nil
		}
		if err := os.WriteFile(path, out.Bytes(), 0644); err != nil {
			return err
		}
		cleaned++
		return nil
	})
	if err != nil {
		return cleaned, err
	}
	return cleaned, os.WriteFile(marker, []byte("photos in this directory have had their metadata removed\n"), 0644)
}
