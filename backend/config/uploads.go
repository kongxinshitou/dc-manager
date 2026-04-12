package config

import "os"

var UploadDir = getEnvOrDefault("UPLOAD_DIR", "./uploads")
const MaxUploadSize int64 = 10 << 20 // 10 MB per file
const MaxImagesPerInspection = 9

var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
