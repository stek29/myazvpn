package dnsmap

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	wg  sync.WaitGroup
	udp *dns.Server
	tcp *dns.Server

	mux *dns.ServeMux

	cli      *dns.Client
	upstream string

	mapper IPMapper

	logQ bool
	logA bool
}

// ServerConfig is the configuration for server
type ServerConfig struct {
	// Addr is the binding address
	Addr string
	// Listen on UDP
	UDP bool
	// Listen on TCP
	TCP bool
	// upstreamNet is dns.Client{}.Net to use when forwarding to upstream
	UpstreamNet string
	// upstream is the address of upstream to forward requests to
	Upstream string
	// mapper is the IPv4 remapper to use
	Mapper IPMapper

	// LogQuery enables request logging
	LogQuery bool
	// LogAnswer enables response logging
	LogAnswer bool
}

// NewServer creates a new Server
func NewServer(conf *ServerConfig) *Server {
	s := &Server{
		mux:      dns.NewServeMux(),
		upstream: conf.Upstream,
		cli: &dns.Client{
			Net: conf.UpstreamNet,
		},
		mapper: conf.Mapper,
		logQ:   conf.LogQuery,
		logA:   conf.LogAnswer,
	}

	s.mux.Handle(".", s)

	if conf.TCP {
		s.tcp = s.makeServer(conf.Addr, "tcp")
	}
	if conf.UDP {
		s.udp = s.makeServer(conf.Addr, "udp")
	}

	return s
}

// Start runs Server on both udp/tcp ports and blocks until
// Stop is called, or both servers shut down
func (s *Server) Start() error {
	if s.udp != nil {
		s.wg.Add(1)
		go s.run(s.udp)
	}

	if s.tcp != nil {
		s.wg.Add(1)
		go s.run(s.tcp)
	}

	log.Println("inf: Server started")
	s.wg.Wait()
	log.Println("inf: Server stopped")

	return nil
}

// Shutdown shuts down both servers
func (s *Server) Shutdown(ctx context.Context) {
	_ = s.udp.ShutdownContext(ctx)
	_ = s.tcp.ShutdownContext(ctx)
}

func (s *Server) makeServer(addr, net string) *dns.Server {
	return &dns.Server{
		Addr:    addr,
		Net:     net,
		Handler: s.mux,
	}
}

func (s *Server) run(sub *dns.Server) {
	log.Printf("inf: listening on %s %s\n", sub.Net, sub.Addr)
	if err := sub.ListenAndServe(); err != nil {
		log.Printf("err: error in %s Server: %s\n", sub.Net, err)
	}
	s.wg.Done()
}

func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	q := r.Question[0]
	qname := q.Name
	qtype := dns.Type(q.Qtype)

	if s.logQ {
		log.Printf("inf: got request %v %v\n", qtype, qname)
	}

	// Filter out IXFR/AXFR queries
	if q.Qtype == dns.TypeIXFR || q.Qtype == dns.TypeAXFR {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeNotImplemented)
		w.WriteMsg(m)
		return
	}

	// HTTPS qtype is not supported, and is actively used by
	// Apple devices, which causes them to escape dnsmap
	if q.Qtype == dns.TypeHTTPS {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeNotImplemented)
		w.WriteMsg(m)
		return
	}

	// Return empty responses for AAAA queries
	if q.Qtype == dns.TypeAAAA {
		if s.logA {
			log.Printf("inf: returning empty response for AAAA req: %v\n", qname)
		}
		m := new(dns.Msg)
		m.SetReply(r)
		w.WriteMsg(m)
		return
	}

	ret, _, err := s.cli.Exchange(r, s.upstream)
	if err != nil {
		log.Printf("err: upstream exchange failed for req %v %v: %v\n", qtype, qname, err)

		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	if q.Qtype == dns.TypeA {
		if s.logA {
			log.Printf("inf: remapping addresses for %v %v\n", qtype, qname)
		}

		err := s.RemapAddresses(ret)

		if err != nil {
			log.Printf("err: remap failed for %v %v: %v\n", qtype, qname, err)

			m := new(dns.Msg)
			m.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(m)
			return
		}
	}

	if s.logA {
		log.Printf("inf: got response for %v %v: %v\n", qtype, qname, ret)
	}

	w.WriteMsg(ret)
}

func (s *Server) RemapAddresses(m *dns.Msg) error {
	for _, rr := range m.Answer {
		a, ok := rr.(*dns.A)
		if !ok {
			continue
		}

		ip, dur, err := s.mapper.RemapIP(a.A)
		if err != nil {
			return fmt.Errorf("Remap failed for %v: %w", a.A, err)
		}

		a.A = ip.To4()
		if dur != 0 {
			newTTL := uint32(dur / time.Second)
			if newTTL < a.Hdr.Ttl {
				a.Hdr.Ttl = newTTL
			}
		}
	}

	return nil
}
