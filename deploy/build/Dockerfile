ARG REGISTRY=docker.io
FROM ${REGISTRY}/golang:1.22.2-alpine3.19 AS builder
RUN set -eux
# && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

ARG APP_RELATIVE_PATH=./cmd/server
ENV APP_RELATIVE_PATH=${APP_RELATIVE_PATH}

COPY .. /data/app
WORKDIR /data/app

RUN rm -rf /data/app/bin/
RUN go mod tidy && go build -ldflags="-s -w" -o ./bin/server ${APP_RELATIVE_PATH}
RUN mv config /data/app/bin/


FROM ${REGISTRY}/alpine:3.19
RUN set -eux
#&& sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories


RUN apk add tzdata && cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime \
    && echo "America/Sao_Paulo" > /etc/timezone \
    && apk del tzdata


ARG APP_ENV=prod
ENV APP_ENV=${APP_ENV}

ARG APP_CONF=./config/prod.yml
ENV APP_CONF=${APP_CONF}

ARG GIN_MODE=release
ENV GIN_MODE=${GIN_MODE}

WORKDIR /data/app
COPY --from=builder /data/app/bin /data/app

EXPOSE 8000
ENTRYPOINT [ "./server" ]

#docker build -t  1.1.1.1:5000/demo-api:v1 --build-arg APP_CONF=config/prod.yml --build-arg  APP_RELATIVE_PATH=./cmd/server/...  .
#docker run -it --rm --entrypoint=ash 1.1.1.1:5000/demo-api:v1