package zipper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestZip(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, path string)
		expectError bool
		errorMsg    string
	}{
		{
			name: "zip single file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				testFile := filepath.Join(dir, "test.txt")
				if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
					t.Fatal(err)
				}
				return testFile
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: false,
		},
		{
			name: "zip directory with multiple files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				files := []string{"file1.txt", "file2.txt", "file3.txt"}
				for _, f := range files {
					if err := os.WriteFile(filepath.Join(dir, f), []byte("content "+f), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return dir
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: false,
		},
		{
			name: "zip nested directory structure",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				if err := os.Mkdir(subdir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: false,
		},
		{
			name: "zip empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: false,
		},
		{
			name: "invalid path - dot",
			setup: func(t *testing.T) string {
				return "."
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: true,
			errorMsg:    "invalid path",
		},
		{
			name: "invalid path - double dot",
			setup: func(t *testing.T) string {
				return ".."
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: true,
			errorMsg:    "invalid path",
		},
		{
			name: "nonexistent path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			cleanup:     func(t *testing.T, path string) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inPath := tt.setup(t)
			defer tt.cleanup(t, inPath)

			zipPath, err := Zip(inPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Clean up zip file after test
			defer os.Remove(zipPath)

			// Verify zip file was created
			if _, err := os.Stat(zipPath); os.IsNotExist(err) {
				t.Errorf("zip file was not created at %s", zipPath)
			}

			// Verify zip file can be opened and contains expected files
			verifyZipContents(t, zipPath, inPath)
		})
	}
}

func TestZipFileNaming(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "myfile.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath, err := Zip(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(zipPath)

	expectedName := "myfile.txt.zip"
	if filepath.Base(zipPath) != expectedName {
		t.Errorf("expected zip name %q, got %q", expectedName, filepath.Base(zipPath))
	}
}

func TestZipCleanupOnError(t *testing.T) {
	// This test verifies that incomplete zip files are cleaned up on error
	// We test this by ensuring the zip file doesn't exist after a failed compression

	// Test 1: Path that doesn't exist should not leave a zip file
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist-dir")
	expectedZipPath := "does-not-exist-dir.zip"

	_, err := Zip(nonExistentPath)
	if err == nil {
		os.Remove(expectedZipPath)
		t.Fatal("expected error for nonexistent path")
	}

	// Verify no zip file was created or it was cleaned up
	if _, statErr := os.Stat(expectedZipPath); statErr == nil {
		os.Remove(expectedZipPath)
		t.Error("zip file should not exist after failed compression")
	}

	// Test 2: Verify successful compression does NOT delete the zip file
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath, err := Zip(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(zipPath)

	// Verify file exists after successful completion
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Error("zip file should exist after successful compression")
	}
}

func verifyZipContents(t *testing.T, zipPath, originalPath string) {
	t.Helper()

	// Open the zip file
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip file: %v", err)
	}
	defer zr.Close()

	// Collect original files
	originalFiles := make(map[string][]byte)
	err = filepath.WalkDir(originalPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		originalFiles[path] = content
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk original directory: %v", err)
	}

	// Verify each file in the zip
	for _, f := range zr.File {
		originalContent, exists := originalFiles[f.Name]
		if !exists {
			t.Errorf("unexpected file in zip: %s", f.Name)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			t.Errorf("failed to open file in zip %s: %v", f.Name, err)
			continue
		}

		zipContent, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Errorf("failed to read file in zip %s: %v", f.Name, err)
			continue
		}

		if string(zipContent) != string(originalContent) {
			t.Errorf("content mismatch for %s", f.Name)
		}

		delete(originalFiles, f.Name)
	}

	// Check if any original files were not included in the zip
	if len(originalFiles) > 0 {
		for path := range originalFiles {
			t.Errorf("file not found in zip: %s", path)
		}
	}
}
