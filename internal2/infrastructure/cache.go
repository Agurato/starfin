package infrastructure

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type Cache struct {
	cachePath string
}

// NewCache initializes the cache folder
func NewCache(cachePath string) *Cache {
	err := os.MkdirAll(cachePath, 0755)
	if err != nil {
		log.WithField("error", err).Fatalln("Could not create cache directory")
	}
	cachePath, err = filepath.Abs(cachePath)
	if err != nil {
		log.WithField("error", err).Fatalln("Could not create cache directory")
	}
	log.WithField("path", cachePath).Infoln("Using cache directory")

	return &Cache{
		cachePath: cachePath,
	}
}

// getCachedPath returns the full path from a filepath in the cache
func (c Cache) getCachedPath(filePath string) string {
	return filepath.Join(c.cachePath, filePath)
}

// CachePoster caches a poster from a source URL and the unique key for this poster
func (c Cache) CachePoster(sourceUrl, key string) (hasToWait bool, err error) {
	return c.CacheFile(sourceUrl, "poster"+key)
}

// CachePoster caches a backdrop from a source URL and the unique key for this backdrop
func (c Cache) CacheBackdrop(sourceUrl, key string) (hasToWait bool, err error) {
	return c.CacheFile(sourceUrl, "backdrop"+key)
}

// CachePoster caches a photo from a source URL and the unique key for this photo
func (c Cache) CachePhoto(sourceUrl, key string) (hasToWait bool, err error) {
	return c.CacheFile(sourceUrl, "photo"+key)
}

// CacheFile caches a file from a sourceUrl to the filePath in the cache folder
// Returns true if the URL returns a Status TooManyRequests (429) and will retry at a later moment
// Returns false if the file was immediately cached
func (c Cache) CacheFile(sourceUrl string, filePath string) (hasToWait bool, err error) {
	// Create directories in the requested path if needed
	parent := c.getCachedPath(filepath.Dir(filePath))
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
	if resp.StatusCode == http.StatusTooManyRequests {
		waitSeconds, err := strconv.Atoi(resp.Header.Get("retry-after"))
		if err != nil {
			waitSeconds = 300 // Wait 5 minutes by default
		}
		time.AfterFunc(time.Duration(waitSeconds)*time.Second, func() {
			c.CacheFile(sourceUrl, filePath)
		})
		return true, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, errors.New("could not fetch source file")
	}
	// Write file
	out, err := os.Create(c.getCachedPath(filePath))
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

// isCached returns true if a filepath is in the cache
func (c Cache) isCached(filePath string) bool {
	_, err := os.Stat(c.getCachedPath(filePath))
	return err == nil
}

// getCachedFile returns a buffer to the cached file
func (c Cache) getCachedFile(filePath string) ([]byte, error) {
	return os.ReadFile(c.getCachedPath(filePath))
}
