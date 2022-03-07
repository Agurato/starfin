package media

type Cast struct {
	Character string
	ActorID   int64
}

type Actor struct {
	TMDBID int64
	Name   string
	Photo  string
}
