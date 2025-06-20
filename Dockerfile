FROM golang:latest

WORKDIR /app
COPY main.go .
RUN go mod init pod-server
RUN go mod tidy
RUN go mod download
EXPOSE 8080
CMD ["go", "run", "main.go"]