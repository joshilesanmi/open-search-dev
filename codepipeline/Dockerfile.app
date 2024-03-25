FROM golang:1.19 AS build

ARG GIT_COMMIT
WORKDIR /open-search-dev
COPY . .

RUN go build -ldflags "-X main.revision=$GIT_COMMIT" -o /out/open-search-dev ./clicmd


FROM ubuntu:20.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl

RUN useradd deploy
USER deploy

COPY --from=build /out/open-search-dev /

EXPOSE 8081
