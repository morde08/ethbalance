FROM golang:1.20.2

COPY . /go/src/github.com/morde08/ethbalance
#RUN cd /go/src/github.com/morde08/ethbalance && go get
#RUN go install github.com/morde08/ethbalance
WORKDIR /go/src/github.com/morde08/ethbalance
RUN go mod download
RUN go build -o /ethbalance

ENV GETH http://eth:8545
ENV PORT 9015

RUN mkdir /app
WORKDIR /app
ADD addresses.txt /app

EXPOSE 9015

ENTRYPOINT ["/ethbalance"]
