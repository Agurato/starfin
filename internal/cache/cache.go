package cache

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Agurato/starfin/internal/context"
	log "github.com/sirupsen/logrus"
)

var cachePath string

func InitCache() {
	cachePath = os.Getenv(context.EnvCachePath)
	if cachePath == "" {
		cachePath = "./cache"
	}
	err := os.MkdirAll(cachePath, 0755)
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

func CacheFile(sourceUrl string, filePath string) (hasToWait bool, err error) {
	// Create directories in the requested path if needed
	parent := GetCachedPath(filepath.Dir(filePath))
	if _, err := os.Stat(parent); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(parent, 0755)
		if err != nil {
			return false, err
		}
	}
	// Get file as buffer
	resp, err := http.Get(sourceUrl)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 429 {
		waitSeconds, err := strconv.Atoi(resp.Header.Get("retry-after"))
		if err != nil {
			waitSeconds = 300 // Wait 5 minutes by default
		}
		time.AfterFunc(time.Duration(waitSeconds)*time.Second, func() {
			CacheFile(sourceUrl, filePath)
		})
		return true, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, errors.New("could not fetch source file")
	}
	// Write file
	out, err := os.Create(GetCachedPath(filePath))
	if err != nil {
		return false, err
	}
	defer out.Close()
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return false, err
	}
	log.WithFields(log.Fields{"url": sourceUrl, "size": n}).Debugln("Cached file")

	return false, nil
}

func IsCached(filePath string) bool {
	_, err := os.Stat(GetCachedPath(filePath))
	return err == nil
}

func GetCachedFile(filePath string) ([]byte, error) {
	return os.ReadFile(GetCachedPath(filePath))
}
