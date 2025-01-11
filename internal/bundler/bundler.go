package bundler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

func CreateBuildContextFromTar(tarFileDir string) (bytes.Buffer, error) {
	gzipFile, err := os.Open(tarFileDir)
	if err != nil {
		return bytes.Buffer{}, err
	}

	defer gzipFile.Close()
	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return bytes.Buffer{}, err
	}
	defer gzipReader.Close()

	var tarBuffer bytes.Buffer
	_, err = io.Copy(&tarBuffer, gzipReader)
	if err != nil {
		return bytes.Buffer{}, err
	}

	return tarBuffer, nil
}

func Gzip(sourceDir, outputFile string) error {
	if _, err := os.Stat(outputFile); err == nil {
		if err := os.Remove(outputFile); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("could not schedule output file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	err = filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, file)
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		fileHandle, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileHandle.Close()

		_, err = io.Copy(tarWriter, fileHandle)
		return err
	})

	if err != nil {
		return fmt.Errorf("error walking the directory: %w", err)
	}

	return nil
}

func GzipToReader(sourceDir string) (*os.File, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar.gz", filepath.Base(sourceDir)))
	if err := Gzip(sourceDir, tmp); err != nil {
		return nil, err
	}
	return os.Open(tmp)
}

func Extract(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open gz file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to schedule gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		targetPath := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to schedule directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to schedule parent directories: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to schedule file: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type: %v", header.Typeflag)
		}
	}

	return nil
}

func WriteToPath(r io.Reader, path string) error {
	dir := filepath.Dir(path)
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = io.Copy(fi, r)
	if err != nil {
		return err
	}
	return nil
}

func ExtractSingleTar(filePath, originalPath string, r io.ReadCloser) (io.Reader, error) {
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == filePath || header.Name == originalPath || filepath.Base(header.Name) == originalPath {
			return tarReader, nil
		}
	}
	return nil, errors.New("file not found: " + filePath)
}
