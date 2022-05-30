package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stek29/myazvpn/dnsmap/pkg/dnsmap"
)

var (
	upstream    = flag.String("upstream", "8.8.8.8:53", "addr:port of upstream server")
	upstreamNet = flag.String("upstream-net", "tcp", "net proto of upstream server")

	bind = flag.String("bind", "127.0.0.4:53", "addr:port to bind to")
	udp  = flag.Bool("udp", true, "listen on udp")
	tcp  = flag.Bool("tcp", true, "listen on tcp")
)

func waitForSigShutdown(srv *dnsmap.Server) {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Printf("inf: signal (%s) received, stopping\n", s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	server := dnsmap.NewServer(*bind, *udp, *tcp, *upstreamNet, *upstream, dnsmap.NoopMapper{})

	go waitForSigShutdown(server)
	if err := server.Start(); err != nil {
		log.Printf("err: server error: %v\n", err)
	}

	log.Println("inf: goodbye")
}
