package dnsheaven

import (
	"sync"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config  *Config
	servers []*dns.Server
}

func NewServer(config *Config, resolver Resolver) *Server {
	resolve := func(net string) dns.HandlerFunc {
		return func(r dns.ResponseWriter, msg *dns.Msg) {
			result, err := resolver.Resolve(net, msg)

			if err != nil {
				logrus.WithError(err).WithField("req", msg).Error("error resolving request")

				r.WriteMsg(&dns.Msg{
					MsgHdr: dns.MsgHdr{
						Id:                 msg.Id,
						Response:           true,
						Opcode:             msg.Opcode,
						Authoritative:      false,
						Truncated:          false,
						RecursionDesired:   false,
						RecursionAvailable: false,
						Zero:               false,
						AuthenticatedData:  false,
						CheckingDisabled:   false,
						Rcode:              dns.RcodeServerFailure,
					},
				})

				return
			}

			r.WriteMsg(result)
		}
	}

	servers := make([]*dns.Server, 0, len(config.Address)*2)
	for _, addr := range config.Address {
		servers = append(servers, &dns.Server{
			Addr:    addr,
			Net:     "tcp",
			Handler: resolve("tcp"),
		})

		servers = append(servers, &dns.Server{
			Addr:    addr,
			Net:     "udp",
			Handler: resolve("udp"),
		})
	}

	return &Server{
		config:  config,
		servers: servers,
	}
}

func (s *Server) Start() error {
	wg := &sync.WaitGroup{}

	errch := make(chan error)

	wg.Add(len(s.servers))
	for _, serv := range s.servers {
		go s.runServer(wg, errch, serv)
	}

	go func() {
		wg.Wait()
		close(errch)
	}()

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Shutdown() error {
	errs := make([]error, len(s.servers))
	for i, serv := range s.servers {
		errs[i] = serv.Shutdown()
	}

	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) runServer(wg *sync.WaitGroup, err chan<- error, server *dns.Server) {
	err <- server.ListenAndServe()
	wg.Done()
}
