package model

import (
	"html/template"
)

type MediaInfoJSONOutput struct {
	CreatingLibrary struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"creatingLibrary"`
	Media struct {
		Ref   string              `json:"@ref"`
		Track []map[string]string `json:"track"`
	} `json:"media"`
}

type VideoInfo struct {
	CodecID    string
	Profile    string
	Resolution string
	FrameRate  string
	BitDepth   string
}

type AudioInfo struct {
	CodecID      string
	Channels     string
	Language     string
	SamplingRate string
}

type SubsInfo struct {
	CodecID  string
	Language string
	Forced   string
}

type MediaInfo struct {
	FullOutput template.HTML

	Format     string
	FileSize   string
	Duration   string
	Resolution string
	Video      []VideoInfo
	Audio      []AudioInfo
	Subs       []SubsInfo
}
