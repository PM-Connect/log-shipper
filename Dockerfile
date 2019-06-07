FROM node:10.8 as webui

RUN mkdir -p /src/webui

WORKDIR /src/webui

COPY ./web/ui/ .

RUN yarn install && \
    yarn run build --mode production --modern

FROM golang:1.12-alpine as gobuild

RUN apk add --no-cache alpine-sdk protobuf ca-certificates git && \
    update-ca-certificates

WORKDIR /app
COPY . .

COPY --from=webui /src/webui/dist/ /app/web/ui/dist/

ENV GO111MODULE=on

RUN go get -u github.com/gobuffalo/packr/packr && \
    go get -u github.com/golang/protobuf/protoc-gen-go

RUN make binary

FROM alpine:3.9

COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuild /app/dist/log-shipper /usr/local/bin/

RUN chmod +x /usr/local/bin/log-shipper

EXPOSE 80
VOLUME ["/tmp"]

ENTRYPOINT ["/usr/local/bin/log-shipper"]
