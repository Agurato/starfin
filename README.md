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
    -e COOKIE_SECRET=secret
    -e DB_URL=mongo
    -e DB_PORT=27017
    -e DB_NAME=starfin
    -e DB_USER=
    -e DB_PASSWORD=
    -e TMDB_API_KEY=
    --name starfin 
    -d starfin 
    --host mongo
``` 

### Docker-compose

```yml
services:
  mongo:
    container_name: mongo
    image: mongo
    ports:
      - 27017:27017
    env_file:
      - ./environment/mongo.env
    networks:
      - servers
    volumes:
      - mongo-data:/data/db
    restart: unless-stopped

  starfin:
    container_name: starfin
    image: starfin
    ports:
      - 8081:8080
    environment:
      - DB_URL=mongo
      - DB_PORT=27017
      - DB_NAME=starfin-prod
      - DB_USER=
      - DB_PASSWORD=
      - COOKIE_SECRET=
      - TMDB_API_KEY=
    volumes:
      - /video:/video
    networks:
      - servers
    restart: unless-stopped

volumes:
  mongo-data: null

networks:
  servers: null
```