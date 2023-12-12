package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	grpcsl "github.com/dymensionxyz/dymint/settlement/grpc"
	"github.com/dymensionxyz/dymint/settlement/grpc/mockserv"
)

func main() {
	conf := grpcsl.DefaultConfig

	flag.IntVar(&conf.Port, "port", conf.Port, "listening port")
	flag.StringVar(&conf.Host, "host", "0.0.0.0", "listening address")
	flag.Parse()

	//kv := store.NewDefaultKVStore(".", "db", "settlement")
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
