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
}

// NewServer creates a new Server
// addr is the binding address
// udp - listen on udp
// tcp - listen on tcp
// upstreamNet is dns.Client{}.Net to use when forwarding to upstream
// upstream is the upstream to forward requests to
func NewServer(addr string, udp, tcp bool, upstreamNet, upstream string, mapper IPMapper) *Server {
	s := &Server{
		mux:      dns.NewServeMux(),
		upstream: upstream,
		cli: &dns.Client{
			Net: upstreamNet,
		},
		mapper: mapper,
	}

	s.mux.Handle(".", s)

	if tcp {
		s.tcp = s.makeServer(addr, "tcp")
	}
	if udp {
		s.udp = s.makeServer(addr, "udp")
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
	log.Printf("inf: got request %v %v\n", qtype, qname)

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
		log.Printf("inf: returning empty response for AAAA req: %v\n", qname)
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
		log.Printf("inf: remapping addresses for %v %v\n", qtype, qname)
		err := s.RemapAddresses(ret)

		if err != nil {
			log.Printf("inf: remap failed for %v %v: %v\n", qtype, qname, err)

			m := new(dns.Msg)
			m.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(m)
			return
		}
	}

	log.Printf("inf: got response for %v %v: %v\n", qtype, qname, ret)
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
			a.Hdr.Ttl = uint32(dur / time.Second)
		}
	}

	return nil
}
