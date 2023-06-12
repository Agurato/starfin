package model

type RarbgCategory string

const (
	RarbgCatEbooks          RarbgCategory = "ebooks"
	RarbgCatGamesPCIso      RarbgCategory = "games_pc_iso"
	RarbgCatGamesPCRip      RarbgCategory = "games_pc_rip"
	RarbgCatGamesPS3        RarbgCategory = "games_ps3"
	RarbgCatGamesPS4        RarbgCategory = "games_ps4"
	RarbgCatGamesXbox360    RarbgCategory = "games_xbox360"
	RarbgCatMovies          RarbgCategory = "movies"
	RarbgCatMoviesBDFull    RarbgCategory = "movies_bd_full"
	RarbgCatMoviesBDRemux   RarbgCategory = "movies_bd_remux"
	RarbgCatMoviesX264      RarbgCategory = "movies_x264"
	RarbgCatMoviesX2643D    RarbgCategory = "movies_x264_3d"
	RarbgCatMoviesX2644K    RarbgCategory = "movies_x264_4k"
	RarbgCatMoviesX264720   RarbgCategory = "movies_x264_720"
	RarbgCatMoviesX265      RarbgCategory = "movies_x265"
	RarbgCatMoviesX2654K    RarbgCategory = "movies_x265_4k"
	RarbgCatMoviesX2654KHDR RarbgCategory = "movies_x265_4k_hdr"
	RarbgCatMoviesXvid      RarbgCategory = "movies_xvid"
	RarbgCatMoviesXvid720   RarbgCategory = "movies_xvid_720"
	RarbgCatMusicFlac       RarbgCategory = "music_flac"
	RarbgCatMusicMp3        RarbgCategory = "music_mp3"
	RarbgCatSoftwarePCIso   RarbgCategory = "software_pc_iso"
	RarbgCatTV              RarbgCategory = "tv"
	RarbgCatTVSD            RarbgCategory = "tv_sd"
	RarbgCatTVUHD           RarbgCategory = "tv_uhd"
	RarbgCatXXX             RarbgCategory = "xxx"
)

var (
	torznabIDs = map[RarbgCategory]int{
		RarbgCatEbooks:          7000,
		RarbgCatGamesPCIso:      4050,
		RarbgCatGamesPCRip:      4050,
		RarbgCatGamesPS3:        1080,
		RarbgCatGamesPS4:        1180,
		RarbgCatGamesXbox360:    1050,
		RarbgCatMovies:          2000,
		RarbgCatMoviesBDFull:    2050,
		RarbgCatMoviesBDRemux:   2050,
		RarbgCatMoviesX264:      2000,
		RarbgCatMoviesX2643D:    2060,
		RarbgCatMoviesX2644K:    2000,
		RarbgCatMoviesX264720:   2000,
		RarbgCatMoviesX265:      2000,
		RarbgCatMoviesX2654K:    2000,
		RarbgCatMoviesX2654KHDR: 2000,
		RarbgCatMoviesXvid:      2000,
		RarbgCatMoviesXvid720:   2000,
		RarbgCatMusicFlac:       3040,
		RarbgCatMusicMp3:        3010,
		RarbgCatSoftwarePCIso:   4000,
		RarbgCatTV:              5000,
		RarbgCatTVSD:            5030,
		RarbgCatTVUHD:           5045,
		RarbgCatXXX:             6000,
	}
)

func GetTorznabID(cat RarbgCategory) int {
	if id, ok := torznabIDs[cat]; ok {
		return id
	}
	return 0
}
