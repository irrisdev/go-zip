package zipper

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Zip(inPath string) (string, error) {

	// short validation on path
	inPath = filepath.Clean(inPath)
	if inPath == "." || inPath == ".." {
		return "", errors.New("invalid path")
	}

	dstPath := filepath.Base(inPath)
	if dstPath == "" || dstPath == "." || dstPath == ".." {
		return "", errors.New("invalid path")
	}

	dstPath = fmt.Sprintf("%s.zip", dstPath)

	// collect all files in the path recursivley
	files := make([]string, 0)
	if err := filepath.WalkDir(inPath, func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	}); err != nil {
		return "", err
	}

	// create new file
	zipFile, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}

	// defer outfile closing
	completed := false
	defer func() {
		zipFile.Close()
		if !completed {
			os.Remove(dstPath)
		}
	}()

	// create new zip writer
	zipw := zip.NewWriter(zipFile)

	if err := func() error {
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(inPath, file)
			if err != nil {
				f.Close()
				return err
			}

			zw, err := zipw.Create(relPath)
			if err != nil {
				f.Close()
				return err
			}

			if _, err := io.Copy(zw, f); err != nil {
				f.Close()
				return err
			}

			f.Close()
		}
		return zipw.Close()
	}(); err != nil {
		return "", err
	}

	completed = true

	return dstPath, nil
}

// Source - https://stackoverflow.com/a
// Posted by Astockwell, modified by community. See post 'Timeline' for change history
// Retrieved 2026-01-06, License - CC BY-SA 4.0

// func Unzip(src, dest string) error {
// 	r, err := zip.OpenReader(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		if err := r.Close(); err != nil {
// 			panic(err)
// 		}
// 	}()

// 	os.MkdirAll(dest, 0755)

// 	// Closure to address file descriptors issue with all the deferred .Close() methods
// 	extractAndWriteFile := func(f *zip.File) error {
// 		rc, err := f.Open()
// 		if err != nil {
// 			return err
// 		}
// 		defer func() {
// 			if err := rc.Close(); err != nil {
// 				panic(err)
// 			}
// 		}()

// 		path := filepath.Join(dest, f.Name)

// 		// Check for ZipSlip (Directory traversal)
// 		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
// 			return fmt.Errorf("illegal file path: %s", path)
// 		}

// 		if f.FileInfo().IsDir() {
// 			os.MkdirAll(path, f.Mode())
// 		} else {
// 			os.MkdirAll(filepath.Dir(path), f.Mode())
// 			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
// 			if err != nil {
// 				return err
// 			}
// 			defer func() {
// 				if err := f.Close(); err != nil {
// 					panic(err)
// 				}
// 			}()

// 			_, err = io.Copy(f, rc)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	}

// 	for _, f := range r.File {
// 		err := extractAndWriteFile(f)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
