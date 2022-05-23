package tar

import (
	"archive/tar"
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	//gzip "github.com/klauspost/pgzip"
	"compress/gzip"
)

func TestTarGz(t *testing.T) {
	type File struct {
		Name        string
		ContentSize int64
		UID         int
		Mode        int
	}

	tests := []struct {
		Name  string
		Files []File
	}{
		{
			Name: "simple-test",
			Files: []File{
				{"file1.txt", 1024, 33333, 0644},
				{"file2.txt", 1024, 33333, 0555},
			},
		},
		{
			Name:  "empty-tar",
			Files: []File{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var (
				buf = bytes.NewBuffer(nil)
				gw  = gzip.NewWriter(buf)
				tw  = tar.NewWriter(gw)
			)

			// Write the tar.gz files.
			for _, file := range test.Files {
				err := tw.WriteHeader(&tar.Header{
					Name:     file.Name,
					Size:     file.ContentSize,
					Uid:      file.UID,
					Gid:      file.UID,
					Mode:     int64(file.Mode),
					Typeflag: tar.TypeReg,
				})
				if err != nil {
					t.Fatalf("can not prepare archive: %q", err)
				}
				_, err = tw.Write(make([]byte, file.ContentSize))
				if err != nil {
					t.Fatalf("can not prepare archive: %q", err)
				}
			}
			tw.Flush()
			tw.Close()
			gw.Flush()
			gw.Close()

			// Create the temp directory for uncompressing.
			wd, err := os.MkdirTemp("", "")
			defer os.RemoveAll(wd)
			if err != nil {
				t.Fatalf("can not prepare test: %v", err)
			}
			targetDir := filepath.Join(wd, "target")
			err = os.MkdirAll(targetDir, 0777)
			if err != nil {
				t.Fatalf("cannot extract tar content: %v", err)
			}

			// Extract tar.gz file to temp directory.
			err = ExtractTarGz(buf, targetDir)
			if err != nil {
				t.Fatalf("cannot extract tar.gz content: %v", err)
			}

			// Verify data.
			for _, file := range test.Files {
				stat, err := os.Stat(filepath.Join(targetDir, file.Name))
				if err != nil {
					t.Errorf("expected %s, but get error: %v", file.Name, err)
					continue
				}
				/*
					uid := stat.Sys().(*syscall.Stat_t).Uid
					if uid != uint32(file.UID) {
						t.Errorf("expected uid %d, but get %d", file.UID, uid)
						continue
					}
					gid := stat.Sys().(*syscall.Stat_t).Gid
					if gid != uint32(file.UID) {
						t.Errorf("expected gid %d, but get %d", file.UID, gid)
						continue
					}
				*/

				realMode := stat.Mode()
				expectedMode := fs.FileMode(file.Mode)
				if realMode.String() != expectedMode.String() {
					t.Errorf("expected fileMode %d but returned %d", expectedMode, realMode)
					continue
				}

			}
		})
	}
}

func TestExtracTarGz(t *testing.T) {
	for i := 0; i < 2; i++ {
		var root string
		var err error
		if i == 0 {
			// test the absolute path.
			root, err = os.MkdirTemp("", "")
		} else {
			// test the relative path.
			root, err = os.MkdirTemp(".", "")
		}
		if err != nil {
			t.Fatalf("unable to create temporary dir: %s", root)
		}
		defer os.RemoveAll(root)

		// Prepare the mock workspace data.
		srcDir := filepath.Join(root, "src")
		filePath := filepath.Join(srcDir, "file1.txt")
		cmd := exec.Command("bash", "-c", "mkdir -p "+srcDir+"&&"+" echo \"this is file1.\" >> "+filePath)
		err = cmd.Run()
		if err != nil {
			t.Fatalf("unable to create file %s. error: %v", filePath, err)
		}
		subDir := filepath.Join(srcDir, "file2")
		filePath = filepath.Join(subDir, "file2.txt")
		cmd = exec.Command("bash", "-c", "mkdir -p "+subDir+"&&"+" echo \"this is file2.\" >> "+filePath)
		err = cmd.Run()
		if err != nil {
			t.Fatalf("unable to create file %s. error: %v", filePath, err)
		}

		// Tar the src directory and save it to data.tar.gz
		dataFilePath := filepath.Join(root, "data.tar.gz")
		f, err := os.OpenFile(dataFilePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf("unalbe to create file %s. error: %v", dataFilePath, err)
		}
		TarGz(srcDir, f)

		// Extract data.tar.gz to dst directory.
		dstDir := filepath.Join(root, "dst")
		f, err = os.Open(dataFilePath)
		if err != nil {
			t.Fatalf("unalbe to read file %s. error: %v", dataFilePath, err)
		}
		ExtractTarGz(f, dstDir)
		f.Close()

		// Verify the data.
		cmd = exec.Command("diff", "--recursive", srcDir, dstDir)
		err = cmd.Run()
		if err != nil {
			t.Fatalf("The two directories are not equal.\nSrc dir: %s\nDst dir: %s\nError: %v", srcDir, dstDir, err)
		}
	}
}
