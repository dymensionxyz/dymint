FROM golang:1.23.3-alpine3.19

WORKDIR /app

COPY . .

WORKDIR /app/da/grpc/mockserv/cmd

# Command to run the executable
CMD ["go","run", "main.go"]