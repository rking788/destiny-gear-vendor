FROM golang:1.11 as builder

WORKDIR /go/src/github.com/rking788/destiny-gear-vendor

COPY . .

RUN go get ./cmd/...

RUN cd cmd/gallery && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/gallery && cp gallery.tpl.html screen.css /bin/
RUN cd cmd/server && make linux
RUN cd cmd/texplode && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/texplode

FROM usd-toolset:latest

WORKDIR /root/

## This should really be done by the application if they don't exist but this is a quick fix for now.
RUN mkdir -p ./local_tools/geom/geometry ./local_tools/geom/textures ./output/gear.scnassets

COPY --from=0 /bin/gallery /bin/gallery.tpl.html /bin/screen.css /bin/server /bin/texplode /root/

CMD ["./server"]
