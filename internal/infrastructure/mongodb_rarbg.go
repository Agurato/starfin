package infrastructure

import (
	"context"
	"fmt"
	"strings"

	_ "github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Agurato/starfin/internal/model"
)

func (m *MongoDB) InitRarbg(dbName string) {
	m.rarbgColl = m.client.Database(dbName).Collection("torrents")
}

func (m *MongoDB) SearchTorrents(ctx context.Context, search, category string, page uint) (torrents []model.RarbgTorrent, err error) {
	const limit = 100

	opt := options.Find().
		SetSort(bson.M{"dt": -1}).
		SetSkip(int64(page-1) * limit).
		SetLimit(limit)

	filter := bson.M{}
	if search != "" {
		search = strings.Trim(strings.ToLower(search), " ")
		searchWords := strings.FieldsFunc(search, func(r rune) bool {
			return r == '.' || r == ' '
		})
		var andWords []bson.M
		for _, w := range searchWords {
			andWords = append(andWords, bson.M{"title": primitive.Regex{Pattern: fmt.Sprintf(`^(.*[^a-z0-9])?%s[^a-z0-9].*$`, w), Options: "i"}})
		}
		filter["$and"] = andWords
	}
	if category != "" {
		filter["cat"] = primitive.Regex{Pattern: fmt.Sprintf("^%s.*$", category), Options: "i"}
	}

	torrentsCur, err := m.rarbgColl.Find(ctx, filter, opt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.Error().Err(err).Msg("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}

func (m *MongoDB) GetTorrents(ctx context.Context, imdbID string, offset, limit int64) (torrents []model.RarbgTorrent, err error) {
	opt := options.Find().
		SetSkip(offset).
		SetLimit(limit)

	filter := bson.M{}
	if imdbID == "" {
		filter["imdb"] = bson.M{"$ne": nil}
	} else if strings.HasPrefix(imdbID, "tt") {
		filter["imdb"] = imdbID
	} else {
		filter["imdb"] = fmt.Sprintf("tt%s", imdbID)
	}
	torrentsCur, err := m.rarbgColl.Find(ctx, filter, opt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.Error().Err(err).Msg("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}

func (m *MongoDB) GetAllTVTorrents(ctx context.Context, offset, limit int64) (torrents []model.RarbgTorrent, err error) {
	opt := options.Find().
		SetSkip(offset).
		SetLimit(limit)

	filter := bson.M{"cat": primitive.Regex{Pattern: `tv`, Options: "i"}}

	torrentsCur, err := m.rarbgColl.Find(ctx, filter, opt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.Error().Err(err).Msg("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}

func (m *MongoDB) GetTVTorrents(ctx context.Context, imdbID, season, episode string, offset, limit int64) (torrents []model.RarbgTorrent, err error) {
	opt := options.Find().
		SetSkip(offset).
		SetLimit(limit)

	filter := bson.M{}
	if strings.HasPrefix(imdbID, "tt") {
		filter["imdb"] = imdbID
	} else {
		filter["imdb"] = fmt.Sprintf("tt%s", imdbID)
	}

	titleFilter := []bson.M{
		{"title": primitive.Regex{Pattern: fmt.Sprintf(`s%s`, season), Options: "i"}},
	}
	if episode != "" {
		titleFilter = append(titleFilter, bson.M{"title": primitive.Regex{Pattern: fmt.Sprintf(`e%s`, episode), Options: "i"}})
	}
	filter["$and"] = titleFilter

	torrentsCur, err := m.rarbgColl.Find(ctx, filter, opt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.Error().Err(err).Msg("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}
