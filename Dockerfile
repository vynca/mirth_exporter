FROM golang:1.16-alpine

RUN apk update && apk add curl && apk add --no-cache bash && apk add openjdk8
WORKDIR /app

COPY go.sum ./
COPY .env ./
COPY go.mod ./
RUN go mod download

COPY *.go ./

RUN curl -SL 'https://s3.us-east-1.amazonaws.com/downloads.mirthcorp.com/connect/3.9.1.b263/mirthconnectcli-3.9.1.b263-unix.tar.gz' \
   | tar -xzC /opt \
     && mv "/opt/Mirth Connect CLI" /opt/connect

RUN go build -o /mirth_exporter

EXPOSE 9041

CMD ["/mirth_exporter"]
