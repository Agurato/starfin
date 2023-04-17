FROM golang:1.20.3

# Install dependencies
RUN apt-get update && \
    apt-get install -y mediainfo

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
ENV MEDIAINFO_PATH=/usr/bin/mediainfo
ENV GIN_MODE=release
ENV PORT=8080
ENV CACHE_PATH=/cache
ENV ITEMS_PER_PAGE=60

EXPOSE ${PORT}
CMD [ "./starfin" ]
