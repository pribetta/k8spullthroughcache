FROM golang:1.22-alpine as devenv

WORKDIR /app

COPY . /app/

RUN go mod tidy

RUN CGO_ENABLED=0 go build -o /pullthru

FROM registry.hub.docker.com/library/docker:dind

USER root

COPY --from=devenv /usr/local/go/ /usr/local/go/

COPY --from=devenv /app/ /app

COPY --from=devenv /pullthru /usr/local/bin/pullthru

RUN chmod +x /usr/local/bin/pullthru

WORKDIR /app

ENV PATH="/usr/local/go/bin:${PATH}"

RUN apk add --no-cache aws-cli

ENTRYPOINT ["pullthru"]

