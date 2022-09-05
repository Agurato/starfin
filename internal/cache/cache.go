package cache

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Agurato/starfin/internal/context"
	"github.com/sirupsen/logrus"
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
	// Get file as buffer
	res, err := http.Get(sourceUrl)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("could not fetch source file")
	}
	var buffer []byte
	_, err = res.Body.Read(buffer)
	if err != nil {
		return err
	}
	// Create directories in the requested path if needed
	parent := GetCachedPath(filepath.Dir(filePath))
	if _, err := os.Stat((parent)); err != nil {
		err = os.MkdirAll(parent, os.ModeDir)
		if err != nil {
			return err
		}
	}
	// Write file
	err = os.WriteFile(GetCachedPath(filePath), buffer, 0644)
	if err != nil {
		return err
	}
	logrus.WithField("url", sourceUrl).Debugln("Cached file")

	return nil
}

func IsCached(filePath string) bool {
	_, err := os.Stat(GetCachedPath(filePath))
	return err == nil
}

func GetCachedFile(filePath string) ([]byte, error) {
	return os.ReadFile(GetCachedPath(filePath))
}
