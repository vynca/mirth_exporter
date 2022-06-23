FROM golang:1.16-alpine

RUN apk update && apk add curl && apk add --no-cache bash
WORKDIR /app

COPY go.sum ./
COPY .env ./
COPY go.mod ./
RUN go mod download

COPY *.go ./
# TODO just pull down mirth-cli-launcher instead of entire mirth server, etc.
# running into AWS access denied errors, even locally when logged in through aws-value or sso
RUN #curl -SL 'https://mirth-client-library.s3.amazonaws.com/mirth-cli-launcher.jar'

RUN curl -SL 'https://s3.amazonaws.com/downloads.mirthcorp.com/connect/3.11.0.b2609/mirthconnect-3.11.0.b2609-unix.tar.gz' \
   | tar -xzC /opt \
     && mv "/opt/Mirth Connect" /opt/connect

RUN go build -o /mirth_exporter

EXPOSE 9041

CMD ["/mirth_exporter"]
