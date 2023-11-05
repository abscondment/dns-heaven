package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"

	"github.com/greenboxal/dns-heaven"
	"github.com/greenboxal/dns-heaven/osx"
	"github.com/sirupsen/logrus"
)

var config = &dnsheaven.Config{}

func init() {
	addresses := ""
	flag.StringVar(&addresses, "addresses", "127.0.0.1:53,[::1]:53", "addresses to listen to, comma-delimited")
	flag.IntVar(&config.Timeout, "timeout", 2000, "request timeout")
	flag.IntVar(&config.Interval, "interval", 1000, "interval between requests")


	config.Address = make([]string, 0)
	for _, address := range strings.Split(addresses, ",") {
		if address != "" {
			config.Address = append(config.Address, address)
		}
	}
}

func main() {
	flag.Parse()

	resolver, err := osx.New(config)

	if err != nil {
		logrus.WithError(err).Error("error starting server")
		os.Exit(1)
	}

	server := dnsheaven.NewServer(config, resolver)

	stopping := false
	go func() {
		err := server.Start()

		if !stopping && err != nil {
			logrus.WithError(err).Error("error starting server")
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	_ = <-sig

	server.Shutdown()
}
