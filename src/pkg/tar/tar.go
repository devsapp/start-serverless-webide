package tar

// Reference: https://github.com/mimoo/eureka/blob/master/folders.go

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	//gzip "github.com/klauspost/pgzip"
	"compress/gzip"

	"github.com/golang/glog"
)

// Check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// Extract the tar.gz stream data and write to the local file.
// src is the source of the tar.gz stream
// dst is the destination of the local directory. If dst directory does not exist, then create it.
func ExtractTarGz(src io.Reader, dst string) error {
	if _, err := os.Stat(dst); err != nil {
		// Create the directory if necessary.
		if errors.Is(err, fs.ErrNotExist) {
			if err = os.MkdirAll(dst, 0755); err != nil {
				glog.Errorf("Create directory failed: %s Error: %v", dst, err)
				return err
			}
			glog.Infof("Create directory %s succeeded.", dst)
		}
	}
	uncompressedStream, err := gzip.NewReader(src)
	if err == io.EOF {
		glog.Infof("The source is empty: %+v", src)
		return nil
	} else if err != nil {
		glog.Errorf("New gzip reader %+v failed: %v", src, err)
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		// Reach the end of the stream.
		if err == io.EOF {
			break
		}

		if err != nil {
			glog.Errorf("New tar reader %+v failed: %v", uncompressedStream, err)
			return err
		}

		if !validRelPath(header.Name) {
			glog.Errorf("Tar conained invalid name: %s", header.Name)
			return fmt.Errorf("tar containerd invalid name: %s", header.Name)
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		// If it's a directory and does not exist, then create it with 0755 permission.
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					glog.Errorf("MkdirAll(%s, 0755) failed: %v", target, err)
					return err
				}
			}
		// If it's a file, create it with same permission.
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				glog.Errorf("Open file %s failed. Error: %v", target, err)
				return err
			}
			if _, err := io.Copy(fileToWrite, tarReader); err != nil {
				glog.Errorf("Write file %+v failed. Error: %v", fileToWrite, err)
				return err
			}

			// Manually close here after each file operation. defering would cause each file
			// close to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	return nil
}

// Compress a file or directory as tar.gz and write to the destination io stream.
// src is the source of the file or directory.
// dst is the destination of the io stream.
func TarGz(src string, dst io.Writer) error {
	gzipWriter := gzip.NewWriter(dst)
	tarWriter := tar.NewWriter(gzipWriter)

	fi, err := os.Stat(src)
	if err != nil {
		glog.Errorf("Stat %s failed. Error: %v", src, err)
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() { // handle regular file
		header, err := tar.FileInfoHeader(fi, src)
		if err != nil {
			glog.Errorf("Get %s file info failed: %v", src, err)
			return err
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			glog.Errorf("Write file header failed: %v", err)
			return err
		}
		data, err := os.Open(src)
		if err != nil {
			glog.Errorf("Open file %s failed: %v", src, err)
			return err
		}
		if _, err := io.Copy(tarWriter, data); err != nil {
			glog.Errorf("Write tar failed: %v", err)
			return err
		}
	} else if mode.IsDir() { // handle directory
		filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
			// Generate the tar header.
			header, err := tar.FileInfoHeader(info, path)
			if err != nil {
				glog.Errorf("Get %s file info header failed: %v", path, err)
				return err
			}

			header.Name, err = filepath.Rel(src, path)
			if err != nil {
				glog.Errorf("Get relative path failed. Base path: %s Target path: %s Error: %v", src, path, err)
				return err
			}

			// Write tar header.
			if err := tarWriter.WriteHeader(header); err != nil {
				glog.Errorf("Write tar header failed: %v", err)
				return err
			}

			// Write regular file.
			if !info.IsDir() {
				data, err := os.Open(path)
				if err != nil {
					glog.Errorf("Open %s file failed: %v", path, err)
					return err
				}
				if _, err := io.Copy(tarWriter, data); err != nil {
					glog.Errorf("Write tar stream failed: %v", err)
					return err
				}
			}

			return nil
		})
	} else {
		glog.Errorf("File type not supported: %s", mode.String())
		return fmt.Errorf("unsupported file type: %s", mode.String())
	}

	if err := tarWriter.Close(); err != nil {
		glog.Errorf("Close tar writer failed: %v", err)
		return err
	}

	if err := gzipWriter.Close(); err != nil {
		glog.Errorf("Close gzip writer failed: %v", err)
		return err
	}

	return nil
}
