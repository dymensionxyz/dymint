FROM golang:1.24.0-alpine3.21

WORKDIR /app

COPY . .

WORKDIR /app/da/grpc/mockserv/cmd

# Command to run the executable
CMD ["go","run", "main.go"]