FROM arm64v8/golang:1.23
ENV GOARCH=arm64
ENV GOOS=linux
WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -v -o /app/main main.go
CMD ["/app/main"]