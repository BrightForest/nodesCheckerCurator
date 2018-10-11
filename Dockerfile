FROM golang:1.10
RUN mkdir /curator
ADD . /curator/
WORKDIR /curator
RUN go get "github.com/gorilla/websocket"
RUN go build -o curator .
RUN groupadd -r curator && useradd -r -g curator curator
USER curator
EXPOSE 8080
CMD ["/curator/curator"]