# Starfin

Website for your friends to download your legally-obtained movies 🤫

## Run the server

Create `.env` file

```
COOKIE_SECRET=
DB_URL=
DB_PORT=
DB_NAME=
DB_USER=
DB_PASSWORD=
TMDB_API_KEY=
MEDIAINFO_PATH=
```

Build & run (windows)

```
go build .\cmd\starfin\ && .\starfin.exe
```

# Docker

A Dockerfile is available.

## Building image

- Clone the sources
- Run `docker build . -t starfin`

## Launching container

if you have a `mongo` container launched that is named `mongo`:
```sh
docker run 
    -p 9999:8080 
    -v /mnt/movies:/movies 
    COOKIE_SECRET=
    -e DB_URL=mongo
    -e DB_PORT=27017
    -e DB_NAME=starfin
    -e DB_USER=starfin
    -e DB_PASSWORD=password123
    -e TMDB_API_KEY=apikey
    -e MEDIAINFO_PATH=path_to_mediainfo #TODO put it inside the docker
    --name starfin 
    -d starfin 
    --host mongo
``` 

