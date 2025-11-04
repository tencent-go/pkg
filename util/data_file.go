package util

import (
	"archive/zip"
	"github.com/tencent-go/pkg/errx"
	"bytes"
	"path"
	"strings"
)

type DataFile struct {
	Name string
	Data []byte
	Dir  string
}

func ZipFilesBytes(files []DataFile) ([]byte, errx.Error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, file := range files {
		zipPath := path.Join(file.Dir, file.Name)
		zipPath = strings.TrimPrefix(zipPath, "/")
		zipPath = strings.TrimPrefix(zipPath, "./")

		fh := &zip.FileHeader{
			Name:   zipPath,
			Method: zip.Deflate,
		}

		writer, err := zipWriter.CreateHeader(fh)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}

		_, err = writer.Write(file.Data)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}

	return buf.Bytes(), nil
}
