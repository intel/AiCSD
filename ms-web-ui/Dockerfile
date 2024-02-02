########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

# using alpine based image to address vulnerabilities found by Snyk scans
FROM node:20.9-alpine3.17 as base

# add node_modules binaries to $PATH (eg. `ng`)
ENV PATH /app/node_modules/.bin:$PATH

# install the latest stable chromium for automated testing
RUN apk add --no-cache chromium=112.0.5615.165-r0
ENV CHROME_BIN=/usr/bin/chromium-browser

RUN npm config set https-proxy $https_proxy \
    && npm config set proxy $http_proxy

RUN npm install -g npm@10.3.0

ARG USER=1000
# delete the old 'node' user and create new one with same UID as local user's UID
RUN deluser --remove-home node; \
    adduser -u $USER -D -s /bin/sh node
# use as local user for file permission purposes when mounting
USER $USER:$USER

# set working directory (will be volume mounted)
WORKDIR /app

FROM base as builder
USER root
RUN npm install -g @angular/cli@17.1.2
COPY . .
RUN npm install && ng build --configuration production && ls -al /app/dist
USER $USER:$USER

# Final stage will be a ngingx container with all the compiled JS/HTML/CSS
FROM nginx:1.21-alpine

WORKDIR /app
VOLUME ["/app"]
EXPOSE 80

COPY --from=builder /app/dist .
COPY ./nginx/web-ui.conf /etc/nginx/conf.d/default.conf

CMD [ "/bin/sh", "-c", "envsubst < assets/env.template.js > assets/env.js && exec nginx -g 'daemon off;'" ]
