FROM golang:1.25.1-trixie AS build

WORKDIR /app

COPY landing/ ./

RUN go mod download && go mod verify

RUN CGO_ENABLED=0 go build -o server ./main.go

FROM scratch

WORKDIR /

COPY --from=build /app/server /server

COPY --from=build /app/resources /resources

ENTRYPOINT ["/server"]
