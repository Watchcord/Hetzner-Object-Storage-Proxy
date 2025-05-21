FROM golang:1.22 AS build
WORKDIR /app
COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /server .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
RUN apk add --no-cache curl

ENV GIN_MODE=release
ENV PORT=3000

COPY --from=build /server /server

EXPOSE 3000
CMD ["/server"]
