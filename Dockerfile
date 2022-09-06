FROM golang:1.19

# Source files will be in /starfin
RUN mkdir /starfin
WORKDIR /starfin

# Cache dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download && go mod verify

# Copy sources
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./web ./web

# Build
RUN go build ./cmd/starfin

# Set environment variables
ENV COOKIE_SECRET=cookiesecret 
ENV DB_URL=mongo
ENV DB_PORT=27017
ENV DB_NAME=starfin
ENV DB_USER=starfin
ENV DB_PASSWORD=password123
ENV TMDB_API_KEY=
ENV MEDIAINFO_PATH=
ENV GIN_MODE=release
ENV PORT=8080
ENV CACHE_PATH=/cache

EXPOSE ${PORT}
CMD [ "./starfin" ]