package main

import (
	"context"
	"encoding/json"
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

func usage() {
	flag.PrintDefaults()

	fmt.Println("default config: ")
	data, err := json.MarshalIndent(&defaultConf, "", "  ")
	if err != nil {
		fmt.Printf("cant get default config: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

func readConfig(config *confRoot, path string) error {
	fd, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cant open config: %w", err)
	}
	defer fd.Close()

	err = json.NewDecoder(fd).Decode(config)
	if err != nil {
		return fmt.Errorf("cant parse config: %w", err)
	}

	return nil
}

func (c *confRoot) run() {
	nftTable := c.Remap.NFT.Table
	nftMap := c.Remap.NFT.Map

	innerMapper, err := nftmap.NewNFTMapper(nftTable, nftMap)
	if err != nil {
		log.Fatalf("cant create nftables mapper for map %v in table %v: %v\n", nftMap, nftTable, err)
	}
	defer innerMapper.Close()

	if c.Remap.NFT.Clear {
		if err = innerMapper.Clear(); err != nil {
			log.Fatalf("cant clear nftables map: %v\n", err)
		}
		log.Printf("inf: cleared current nftables map %v in table %v\n", nftMap, nftTable)
	}

	mapper, err := newDNSMapper(c.Remap.Range, innerMapper)
	if err != nil {
		log.Fatalf("cant create mapper for range `%v`: %v\n", c.Remap.Range, err)
	}

	server := dnsmap.NewServer(&dnsmap.ServerConfig{
		Addr: c.Listen.Addr,
		UDP:  c.Listen.UDP,
		TCP:  c.Listen.TCP,

		UpstreamNet: c.Upstream.Net,
		Upstream:    c.Upstream.Addr,

		Mapper: mapper,

		LogQuery:  c.Log.Query,
		LogAnswer: c.Log.Answer,
	})

	go waitForSigShutdown(server)
	if err := server.Start(); err != nil {
		log.Printf("err: server error: %v\n", err)
	}
}

func main() {
	configPath := flag.String("config", "", "path to config.json")

	flag.Usage = usage
	flag.Parse()

	config := defaultConf
	if *configPath == "" {
		log.Println("warn: no config is provided, running with defaults")
	} else {
		if err := readConfig(&config, *configPath); err != nil {
			log.Fatalf("cant read config file `%v`: %v\n", *configPath, err)
		}
	}

	config.run()
	log.Println("inf: goodbye")
}
