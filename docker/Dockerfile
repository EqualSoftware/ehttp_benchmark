# Builds one benchmark app. APP picks the folder under apps/.
# The build needs the ehttp library for the "replace" line of go.mod. Pass it
# as the named context "ehttp" (compose.yaml does this):
#   docker build --build-context ehttp=../ehttp --build-arg APP=gin -f docker/Dockerfile .
FROM golang:1.27.3-alpine AS build
ARG APP
WORKDIR /bench
# go.mod has "replace github.com/EqualSoftware/ehttp => ../ehttp", so ehttp lives at /ehttp.
COPY --from=ehttp / /ehttp
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
