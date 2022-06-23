FROM golang:1.16-alpine

RUN apk update && apk add curl && apk add --no-cache bash
WORKDIR /app

COPY go.sum ./
COPY go.mod ./
RUN go mod download

COPY *.go ./

RUN curl -SL 'https://s3.amazonaws.com/downloads.mirthcorp.com/connect/3.11.0.b2609/mirthconnect-3.11.0.b2609-unix.tar.gz' \
    | tar -xzC /opt \
    && mv "/opt/Mirth Connect" /opt/connect

# TODO keep cli-lib folder, config/mirth_cli_config.properties , & mirth_cli_launcher.jar, everything else can be removed

RUN go build -o /mirth_exporter

EXPOSE 9041

CMD ["/mirth_exporter"]
