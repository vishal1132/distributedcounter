FROM golang:1.16-alpine AS build
WORKDIR /app/
COPY . .
RUN go install
RUN CGO_ENABLED=0 go test ./...
RUN go build -o ./main

FROM alpine:3.13.2
COPY --from=build /app/main main
ENTRYPOINT ["/main"]