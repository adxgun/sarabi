package sarabi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
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

func GzipDirectory(sourceDir, outputFile string) error {
	if _, err := os.Stat(outputFile); err == nil {
		if err := os.Remove(outputFile); err != nil {
			return err
		}
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
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
