package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/grpc/mockserv"
)

func main() {
	conf := settlement.GrpcConfig{
		Host: "127.0.0.1",
		Port: 7981,
	}

	flag.IntVar(&conf.Port, "port", conf.Port, "listening port")
	flag.StringVar(&conf.Host, "host", "0.0.0.0", "listening address")
	flag.Parse()

	lis, err := net.Listen("tcp", conf.Host+":"+strconv.Itoa(conf.Port))
	if err != nil {
		log.Panic(err)
	}
	log.Println("Listening on:", lis.Addr())
	srv := mockserv.GetServer(conf)
	if err := srv.Serve(lis); err != nil {
		log.Println("error while serving:", err)
	}
}
