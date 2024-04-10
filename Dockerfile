FROM golang:1.22-alpine as build
WORKDIR /
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-w -s' -o /lynxgate

FROM alpine:latest
COPY --from=build /lynxgate /
EXPOSE 8080
CMD [ "/lynxgate" ]