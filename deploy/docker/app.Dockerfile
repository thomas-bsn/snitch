FROM node:24.21.0-alpine@sha256:ebfe2f90462722a7a4de65e91990e97fe0d401c70e0e762c5b53302f905ec1c1 AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY web/ ./
COPY scripts/collect_web_notices.mjs /collect_web_notices.mjs
RUN node /collect_web_notices.mjs
RUN npm run build

FROM golang:1.27.3-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS go
WORKDIR /src/app
COPY app/go.mod app/go.sum ./
RUN go mod download
COPY app/ ./
COPY scripts/collect_go_notices.sh /collect_go_notices.sh
RUN sh /collect_go_notices.sh
COPY --from=web /src/app/ui/dist ./ui/dist
ARG TARGETOS=linux
ARG TARGETARCH
RUN GOMAXPROCS=2 CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -p 2 -trimpath -ldflags="-s -w" -o /snitch ./cmd/snitch

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
RUN apk add --no-cache ffmpeg=8.1.2-r0 && adduser -D -u 10001 snitch && mkdir -p /data/media && chown snitch:snitch /data/media
COPY --from=go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go /notices /usr/share/licenses/snitch/third-party
COPY --from=web /notices /usr/share/licenses/snitch/third-party
COPY deploy/licenses/ffmpeg/ /usr/share/licenses/snitch/third-party/ffmpeg/
COPY --from=go /snitch /snitch
COPY LICENSE /usr/share/licenses/snitch/LICENSE
RUN ffmpeg -hide_banner -loglevel error -f lavfi -i 'testsrc2=size=64x64:rate=1:duration=1' -an -c:v mpeg4 -pix_fmt yuv420p /tmp/synthetic-video.mp4 \
 && ffmpeg -hide_banner -loglevel error -f lavfi -i 'anullsrc=r=8000:cl=mono' -t 1 -vn -c:a aac /tmp/synthetic-no-video.mp4 \
 && /snitch probe-smoke /tmp/synthetic-video.mp4 /tmp/synthetic-no-video.mp4 \
 && rm /tmp/synthetic-video.mp4 /tmp/synthetic-no-video.mp4
USER snitch
EXPOSE 8080
ENTRYPOINT ["/snitch"]
CMD ["serve"]
