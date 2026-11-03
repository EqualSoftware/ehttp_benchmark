# Builds one benchmark app. APP picks the folder under apps/.
#   docker build --build-arg APP=gin -f docker/Dockerfile .
FROM golang:1.27.3-alpine AS build
ARG APP
WORKDIR /bench
COPY go.mod go.sum ./
RUN go mod download
COPY apps apps
COPY internal internal
RUN test -d "apps/${APP}" && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/app "./apps/${APP}"

FROM alpine:3.24.3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/app /app
ENV PORT=3000
EXPOSE 3000
ENTRYPOINT ["/app"]
