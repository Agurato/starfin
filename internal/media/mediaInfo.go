package media

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func GetMediaInfo(mediaInfoPath, filePath string) (MediaInfo, error) {
	var mediaInfo MediaInfo
	var mediaInfoJSONOutput MediaInfoJSONOutput

	out, err := exec.Command(mediaInfoPath, filePath).Output()
	if err != nil {
		return mediaInfo, err
	}
	fullOutput := strings.ReplaceAll(string(out), "\r\n", "\n")
	fullOutput = strings.Trim(fullOutput, "\n")
	var fullOutputLines []string
	for _, line := range strings.Split(fullOutput, "\n") {
		if strings.HasPrefix(line, "Complete name") {
			fullOutputLines = append(fullOutputLines, fmt.Sprintf("Name : %s", filepath.Base(strings.Split(line, " : ")[1])))
		} else {
			fullOutputLines = append(fullOutputLines, line)
		}
	}
	mediaInfo.FullOutput = template.HTML(strings.Join(fullOutputLines, "<br>"))

	out, err = exec.Command(mediaInfoPath, "--Output=JSON", filePath).Output()
	if err != nil {
		return mediaInfo, err
	}
	json.Unmarshal(out, &mediaInfoJSONOutput)

	for _, track := range mediaInfoJSONOutput.Media.Track {
		switch track["@type"] {
		case "General":
			mediaInfo.Format = track["Format"]
			totalSeconds, _ := strconv.ParseFloat(track["Duration"], 32)
			hours := int(totalSeconds / 3600)
			minutes := int(totalSeconds/60) % 60
			seconds := int(totalSeconds) % 60
			mediaInfo.Duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
			totalSize, _ := strconv.Atoi(track["FileSize"])
			if totalSize > 1_000_000_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f GB", float64(totalSize)/1_000_000_000)
			} else if totalSize > 1_000_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f MB", float64(totalSize)/1_000_000)
			} else if totalSize > 1_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f KB", float64(totalSize)/1_000)
			}
		case "Video":
			mediaInfo.Video = append(mediaInfo.Video, VideoInfo{
				CodecID:    track["CodecID"],
				Profile:    track["Format_Profile"],
				Resolution: fmt.Sprintf("%sx%s", track["Width"], track["Height"]),
				FrameRate:  track["FrameRate"],
				BitDepth:   track["BitDepth"],
			})
			// Compute resolution on first video stream
			if mediaInfo.Resolution == "" {
				// Switch on the width because movie can have black horizontal bars
				width, _ := strconv.Atoi(track["Width"])
				switch width {
				case 720:
					mediaInfo.Resolution = "480p"
				case 1280:
					mediaInfo.Resolution = "720p"
				case 1920:
					mediaInfo.Resolution = "1080p"
				case 2560:
					mediaInfo.Resolution = "1440p"
				case 3840:
					mediaInfo.Resolution = "4K"
				case 7680:
					mediaInfo.Resolution = "8K"
				}
			}
		case "Audio":
			mediaInfo.Audio = append(mediaInfo.Audio, AudioInfo{
				CodecID:      track["CodecID"],
				Channels:     track["Channels"],
				Language:     track["Language"],
				SamplingRate: track["SamplingRate"],
			})
		case "Text":
			mediaInfo.Subs = append(mediaInfo.Subs, SubsInfo{
				CodecID:  track["CodecID"],
				Language: track["Language"],
				Forced:   track["Forced"],
			})
		}
	}

	return mediaInfo, nil
}
