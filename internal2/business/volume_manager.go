package business

import (
	"errors"
	"fmt"
	"os"

	"github.com/Agurato/starfin/internal2/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VolumeStorer interface {
	GetVolumes() ([]model.Volume, error)
	GetVolumeFromID(id primitive.ObjectID) (volume *model.Volume, err error)
	AddVolume(volume *model.Volume) error
	DeleteVolume(volumeId primitive.ObjectID) error
}

type VolumeMetadataGetter interface {
	CreateFilm(file string, volumeID primitive.ObjectID, subFiles []string) *model.Film
	FetchFilmTMDBID(f *model.Film) error
	UpdateFilmDetails(film *model.Film)
}

type VolumeManager interface {
	GetVolumes() ([]model.Volume, error)
	GetVolume(volumeHexID string) (*model.Volume, error)
	CreateVolume(name, path string, isRecursive bool, mediaType string) error
	DeleteVolume(volumeHexID string) error
}

type VolumeManagerWrapper struct {
	VolumeStorer
	VolumeMetadataGetter
	FilmManager
	*FileWatcher
}

func NewVolumeManagerWrapper(vs VolumeStorer, fw *FileWatcher, fm FilmManager, m VolumeMetadataGetter) *VolumeManagerWrapper {
	return &VolumeManagerWrapper{
		VolumeStorer:         vs,
		VolumeMetadataGetter: m,
		FilmManager:          fm,
		FileWatcher:          fw,
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
	go vmw.scanVolume(volume)

	return nil
}

func (vmw VolumeManagerWrapper) DeleteVolume(volumeHexID string) error {
	volumeId, err := primitive.ObjectIDFromHex(volumeHexID)
	if err != nil {
		return fmt.Errorf("Incorrect volume ID: %w", err)
	}

	return vmw.VolumeStorer.DeleteVolume(volumeId)
}

func (vmw VolumeManagerWrapper) scanVolume(volume *model.Volume) {
	videoFiles, subFiles, err := volume.ListVideoFiles()
	if err != nil {
		log.WithField("volumePath", volume.Path).Warningln("Unable to scan folder for video files")
	}

	log.WithField("volumePath", volume.Path).Debugln("Scanning volume")

	// Worker function
	getFilmsFromFiles := func(files <-chan string, films chan<- *model.Film) {
		for file := range files {
			film := vmw.CreateFilm(file, volume.ID, subFiles)

			// Search ID on TMDB
			if err = vmw.VolumeMetadataGetter.FetchFilmTMDBID(film); err != nil {
				log.WithFields(log.Fields{"file": file, "err": err}).Warningln("Unable to fetch film ID from TMDB")
				film.Title = film.Name
			} else {
				log.WithFields(log.Fields{"tmdb_id": film.TMDBID, "file": file}).Infoln("Found TMDB ID for file")
				// Fill info from TMDB
				vmw.VolumeMetadataGetter.UpdateFilmDetails(film)
			}

			films <- film
		}
	}

	// Init channels
	files := make(chan string, len(videoFiles))
	films := make(chan *model.Film, len(videoFiles))

	// Init workers
	for w := 1; w <= 20; w++ {
		go getFilmsFromFiles(files, films)
	}

	// Get films
	for _, file := range videoFiles {
		files <- file
	}
	close(files)

	for film := range films {
		vmw.FilmManager.AddFilm(film, false)
	}

	// Add file watch to the volume
	vmw.FileWatcher.AddVolume(volume)
}
