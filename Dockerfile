FROM golang:latest

ENV COOKIE_SECRET=cookiesecret \
  DB_URL=mongo\
  DB_PORT=27017\
  DB_NAME=starfin\
  DB_USER=starfin\
  DB_PASSWORD=password123\
  TMDB_API_KEY=\
  MEDIAINFO_PATH=

RUN mkdir starfin
WORKDIR starfin
COPY .env .
COPY ./go.mod ./go.sum ./
RUN go mod download && go mod verify
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./web ./web
RUN go build ./cmd/starfin
