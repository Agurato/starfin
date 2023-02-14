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

type VolumeManager interface {
	GetVolumes() ([]model.Volume, error)
	GetVolume(volumeHexID string) (*model.Volume, error)
	CreateVolume(name, path string, isRecursive bool, mediaType string) error
	DeleteVolume(volumeHexID string) error
}

type VolumeManagerWrapper struct {
	VolumeStorer
}

func NewVolumeManagerWrapper(vs VolumeStorer) *VolumeManagerWrapper {
	return &VolumeManagerWrapper{
		VolumeStorer: vs,
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
	go searchMediaFilesInVolume(volume)

	// Add file watch to the volume
	addFileWatch(volume)

	return nil
}

func (vmw VolumeManagerWrapper) DeleteVolume(volumeHexID string) error {
	volumeId, err := primitive.ObjectIDFromHex(volumeHexID)
	if err != nil {
		return fmt.Errorf("Incorrect volume ID: %w", err)
	}

	return vmw.VolumeStorer.DeleteVolume(volumeId)
}
