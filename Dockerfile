# syntax=docker/dockerfile:1

FROM golang:1.27

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY . .
RUN go mod download

COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o ./turtlecnc

EXPOSE 8080

# Run
CMD ["./turtlecnc", "-s"]
