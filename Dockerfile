# syntax=docker/dockerfile:1

# --- Stage 1: build the SvelteKit frontend (adapter-static -> web/build) ---
# In dev this is supplied by the bind mount; prod must build it so the Go server
# can serve ./web/build/index.html (otherwise "/" 404s).
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Stage 2: build the Go server ---
FROM golang:1.23-alpine AS build
WORKDIR /src
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading
# them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /out/main main.go

# --- Stage 3: minimal runtime image ---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/main ./main
# Static assets served via relative paths in app/router/route.go.
COPY --from=build /src/app/static ./app/static
COPY --from=web /web/build ./web/build
EXPOSE 9000
CMD ["/app/main"]
