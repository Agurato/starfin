package cache

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Agurato/starfin/internal/context"
	log "github.com/sirupsen/logrus"
)

var cachePath string

func InitCache() {
	cachePath = os.Getenv(context.EnvCachePath)
	if cachePath == "" {
		cachePath = "./cache"
	}
	err := os.MkdirAll(cachePath, os.ModeDir)
	if err != nil {
		log.WithField("error", err).Fatalln("Could not create cache directory")
	}
	cachePath, err = filepath.Abs(cachePath)
	if err != nil {
		log.WithField("error", err).Fatalln("Could not create cache directory")
	}
	log.WithField("path", cachePath).Infoln("Using cache directory")
}

func GetCachedPath(filePath string) string {
	return filepath.Join(cachePath, filePath)
}

func CacheFile(sourceUrl string, filePath string) error {
	// Create directories in the requested path if needed
	parent := GetCachedPath(filepath.Dir(filePath))
	if _, err := os.Stat(parent); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(parent, os.ModeDir)
		if err != nil {
			return err
		}
	}
	// Get file as buffer
	resp, err := http.Get(sourceUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("could not fetch source file")
	}
	// Write file
	out, err := os.Create(GetCachedPath(filePath))
	if err != nil {
		return err
	}
	defer out.Close()
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"url": sourceUrl, "size": n}).Debugln("Cached file")

	return nil
}

func IsCached(filePath string) bool {
	_, err := os.Stat(GetCachedPath(filePath))
	return err == nil
}

func GetCachedFile(filePath string) ([]byte, error) {
	return os.ReadFile(GetCachedPath(filePath))
}
