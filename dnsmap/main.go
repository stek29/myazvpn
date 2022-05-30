package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stek29/myazvpn/dnsmap/pkg/dnsmap"
	"github.com/stek29/myazvpn/dnsmap/pkg/nftmap"
)

var (
	upstream    = flag.String("upstream", "8.8.8.8:53", "addr:port of upstream server")
	upstreamNet = flag.String("upstream-net", "tcp", "net proto of upstream server")

	bind = flag.String("bind", "127.0.0.4:53", "addr:port to bind to")
	udp  = flag.Bool("udp", true, "listen on udp")
	tcp  = flag.Bool("tcp", true, "listen on tcp")

	remapRange = flag.String("remap-range", "10.222.0.0/20", "ip range to remap IPs into in CIDR format")
	table      = flag.String("nft-table", "nat", "name of nftables table")
	mapname    = flag.String("nft-map", "dnsmap", "name of nftables map")
	nftclear   = flag.Bool("nft-clear", false, "clear nftables map on strart")
)

func waitForSigShutdown(srv *dnsmap.Server) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Printf("inf: signal (%s) received, stopping\n", s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func newDNSMapper(remapRange string, inner nftmap.NFTMapper) (*nftmap.DNSMapper, error) {
	_, remapCidr, err := net.ParseCIDR(remapRange)
	if err != nil || remapCidr == nil {
		return nil, fmt.Errorf("cant parse remap range: %w", err)
	}

	mapper, err := nftmap.NewDNSMapper(*remapCidr, inner)
	if err != nil {
		return nil, fmt.Errorf("cant create mapper: %w", err)
	}

	return mapper, nil
}

func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	innerMapper, err := nftmap.NewNFTMapper(*table, *mapname)
	if err != nil {
		log.Fatalf("cant create nftables mapper for map %v in table %v: %v\n", *table, *mapname, err)
	}
	defer innerMapper.Close()

	if *nftclear {
		if err = innerMapper.Clear(); err != nil {
			log.Fatalf("cant clear nftables map: %v\n", err)
		}
	}

	mapper, err := newDNSMapper(*remapRange, innerMapper)
	if err != nil {
		log.Fatalf("cant create mapper for range `%v`: %v\n", *remapRange, err)
	}

	server := dnsmap.NewServer(*bind, *udp, *tcp, *upstreamNet, *upstream, mapper)

	go waitForSigShutdown(server)
	if err := server.Start(); err != nil {
		log.Printf("err: server error: %v\n", err)
	}

	log.Println("inf: goodbye")
}
