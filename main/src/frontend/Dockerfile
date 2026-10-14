# syntax=docker/dockerfile:1

# RYZE web client image.
#
# The build context is the repository root, because the Vite project lives at the
# root and includes files from ./main:
#
#   docker build -t ryze-web -f main/src/frontend/Dockerfile .
#
# VITE_API_BASE_URL is a build-time value: Vite inlines it into the bundle, so it
# cannot be changed by editing the container at runtime. It defaults to the
# same-origin path that nginx proxies to the API.

FROM node:26-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html tsconfig.json vite.config.ts ./
COPY main ./main

ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM nginx:1.27-alpine

COPY main/src/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
