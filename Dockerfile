FROM golang:1.23@sha256:7ea4c9dcb2b97ff8ee80a67db3d44f98c8ffa0d191399197007d8459c1453041 AS buildgo

RUN mkdir /app
COPY . /app
WORKDIR /app

RUN CGO_ENABLED=0 go build .

FROM alpine@sha256:56fa17d2a7e7f168a043a2712e63aed1f8543aeafdcee47c58dcffe38ed51099

RUN mkdir /app
COPY --from=buildgo /app/nunc /app/

COPY --from=buildgo /usr/local/go/LICENSE /app/GO-LICENSE
COPY licenses.txt /app/THIRD-PARTY-LICENSES

CMD ["/app/nunc"]

