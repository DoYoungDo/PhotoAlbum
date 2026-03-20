package server

import (
	"archive/zip"
	"io"
	"os"

	"photoalbum/internal/service"
)

func writeZipResponse(w io.Writer, entries []service.DownloadEntry) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, entry := range entries {
		writer, err := zw.Create(entry.FileName)
		if err != nil {
			return err
		}
		file, err := os.Open(entry.Path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return zw.Close()
}
