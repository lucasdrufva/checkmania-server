FROM golang:1.15 as builder
ENV APP_USER app
ENV APP_HOME /go/src/
RUN groupadd $APP_USER && useradd -m -g $APP_USER -l $APP_USER
RUN mkdir -p $APP_HOME && chown -R $APP_USER:$APP_USER $APP_HOME

WORKDIR $APP_HOME
USER $APP_USER
COPY src/ .

RUN go mod download
RUN go mod verify
RUN go build -o server

FROM debian:buster
ENV APP_USER app
ENV APP_HOME /go/src/
RUN groupadd $APP_USER && useradd -m -g $APP_USER -l $APP_USER
RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME
COPY src/conf/ conf/
COPY --chown=0:0 --from=builder $APP_HOME/server $APP_HOME

EXPOSE 8080
USER $APP_USER
CMD ["./server"]