package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Agurato/starfin/internal/model"
	_ "github.com/glebarez/go-sqlite"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		log.WithField("error", err).Errorln("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}

func (m *MongoDB) GetTorrents(ctx context.Context, imdbid string, offset, limit int64) (torrents []model.RarbgTorrent, err error) {
	opt := options.Find().
		SetSkip(offset).
		SetLimit(limit)

	filter := bson.M{}
	if imdbid == "" {
		filter["imdb"] = bson.M{"$ne": nil}
	} else {
		filter["imdb"] = imdbid
	}
	torrentsCur, err := m.rarbgColl.Find(ctx, filter, opt)
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve torrents from database")
		return
	}
	for torrentsCur.Next(m.ctx) {
		var rt model.RarbgTorrent
		err := torrentsCur.Decode(&rt)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch torrent from database")
		}
		torrents = append(torrents, rt)
	}
	return
}
