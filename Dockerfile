FROM golang:1.19-alpine as builder

COPY . /go/src/app

WORKDIR /go/src/app

RUN go build ./cmd/cosmos-notifyer


FROM alpine

COPY --from=builder /go/src/app/cosmos-notifyer /bin/cosmos-notifyer

ENTRYPOINT [ "/bin/cosmos-notifyer" ]
CMD [ "start" ]
