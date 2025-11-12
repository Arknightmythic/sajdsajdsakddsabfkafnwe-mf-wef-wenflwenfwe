package config

import (
	"os"
	"path/filepath"
)

func GetUploadPath() string {
	uploadPath := os.Getenv("UPLOAD_PATH")
	if uploadPath == "" {
		uploadPath = "./uploads/documents"
	}

	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		panic("Failed to create upload directory: " + err.Error())
	}

	return uploadPath
}

func GetDocumentPath(filename string) string {
	return filepath.Join(GetUploadPath(), filename)
}
