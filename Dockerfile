FROM golang:1.16-alpine

RUN apk update && apk add curl && apk add --no-cache bash
WORKDIR /app

COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN curl -SL 'https://s3.amazonaws.com/downloads.mirthcorp.com/connect/3.11.0.b2609/mirthconnect-3.11.0.b2609-unix.tar.gz' \
    | tar -xzC /opt \
    && mv "/opt/Mirth Connect" /opt/connect

COPY /opt/connect/conf/mirth-cli-config.properties mirth-cli-config.properties
COPY /opt/connect/mirth-cli-launcher.jar mirth-cli-launcher.jar
RUN rm -rf /opt/connect

RUN go build -o /mirth_exporter

EXPOSE 9041

CMD ["/mirth_exporter"]
