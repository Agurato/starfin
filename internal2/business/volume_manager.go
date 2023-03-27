package business

import (
	"errors"
	"fmt"
	"os"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/alitto/pond"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VolumeStorer interface {
	GetVolumes() ([]model.Volume, error)
	GetVolumeFromID(id primitive.ObjectID) (volume *model.Volume, err error)
	AddVolume(volume *model.Volume) error
	DeleteVolume(volumeId primitive.ObjectID) error
}

type VolumeManager interface {
	GetVolumes() ([]model.Volume, error)
	GetVolume(volumeHexID string) (*model.Volume, error)
	CreateVolume(name, path string, isRecursive bool, mediaType string) error
	DeleteVolume(volumeHexID string) error
}

type VolumeManagerWrapper struct {
	VolumeStorer
	FilmManager
	*FileWatcher
}

func NewVolumeManagerWrapper(vs VolumeStorer, fw *FileWatcher, fm FilmManager) *VolumeManagerWrapper {
	return &VolumeManagerWrapper{
		VolumeStorer: vs,
		FileWatcher:  fw,
		FilmManager:  fm,
	}
}

func (vmw VolumeManagerWrapper) GetVolumes() ([]model.Volume, error) {
	return vmw.VolumeStorer.GetVolumes()
}

func (vmw VolumeManagerWrapper) GetVolume(volumeHexID string) (*model.Volume, error) {
	volumeId, err := primitive.ObjectIDFromHex(volumeHexID)
	if err != nil {
		return nil, fmt.Errorf("Incorrect volume ID: %w", err)
	}
	volume, err := vmw.VolumeStorer.GetVolumeFromID(volumeId)
	if err != nil {
		return nil, fmt.Errorf("Could not get volume from ID '%s': %w", volumeHexID, err)
	}
	return volume, nil
}

func (vmw VolumeManagerWrapper) CreateVolume(name, path string, isRecursive bool, mediaType string) error {
	volume := &model.Volume{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Path:        path,
		IsRecursive: isRecursive,
		MediaType:   mediaType,
	}

	// Check volume name length
	if len(volume.Name) < 3 {
		return errors.New("volume name must be between 3")
	}

	// Check path is a directory
	fileInfo, err := os.Stat(volume.Path)
	if err != nil {
		return errors.New("volume path does not exist")
	}
	if !fileInfo.IsDir() {
		return errors.New("volume path is not a directory")
	}

	// Add volume to the database
	err = vmw.VolumeStorer.AddVolume(volume)
	if err != nil {
		log.Errorln(err)
		return errors.New("volume could not be added")
	}

	// Search for media files in a separate goroutine to return the page asap
	go func() {
		// Channel to add film to DB as they are fetched from TMDB
		filmChan := make(chan *model.Film)

		go vmw.scanVolume(volume, filmChan)

		for {
			film, more := <-filmChan
			if more {
				vmw.FilmManager.AddFilm(film, false)
			} else {
				break
			}
		}
	}()

	// Add file watch to the volume
	vmw.FileWatcher.AddVolume(volume)

	return nil
}

func (vmw VolumeManagerWrapper) DeleteVolume(volumeHexID string) error {
	volumeId, err := primitive.ObjectIDFromHex(volumeHexID)
	if err != nil {
		return fmt.Errorf("Incorrect volume ID: %w", err)
	}

	return vmw.VolumeStorer.DeleteVolume(volumeId)
}

// scanVolume scan files from volume that have not been added to the db yet
func (vmw VolumeManagerWrapper) scanVolume(v *model.Volume, mediaChan chan *model.Film) {
	videoFiles, subFiles, err := v.ListVideoFiles()
	if err != nil {
		log.WithField("volumePath", v.Path).Warningln("Unable to scan folder for video files")
	}

	log.WithField("volumePath", v.Path).Debugln("Scanning volume")

	// Create worker pool of size 20
	pool := pond.New(20, 0, pond.MinWorkers(20))

	// For each file
	for _, file := range videoFiles {
		file := file
		pool.Submit(func() {
			film := model.NewFilm(file, v.ID, subFiles)

			// Search ID on TMDB
			if err = film.FetchTMDBID(); err != nil {
				log.WithFields(log.Fields{"file": file, "err": err}).Warningln("Unable to fetch film ID from TMDB")
				film.Title = film.Name
			} else {
				log.WithField("tmdbID", film.TMDBID).Infoln("Found media with TMDB ID")
				// Fill info from TMDB
				film.FetchDetails()
			}

			// Send media to the channel
			mediaChan <- film
		})
	}

	pool.StopAndWait()
	close(mediaChan)
}
