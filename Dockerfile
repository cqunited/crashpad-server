FROM golang:alpine3.13
WORKDIR $GOPATH/src/github.com/riclava/crashpad-server
COPY . $GOPATH/src/github.com/riclava/crashpad-server
RUN go build .
EXPOSE 8080
ENTRYPOINT ["./crashpad-server"]
