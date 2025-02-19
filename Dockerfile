FROM golang:1.23.6-alpine3.20

WORKDIR /app

COPY . .

WORKDIR /app/da/grpc/mockserv/cmd

# Command to run the executable
CMD ["go","run", "main.go"]