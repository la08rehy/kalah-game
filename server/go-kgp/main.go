package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

const (
	majorVersion = 1
	minorVersion = 0
	patchVersion = 0

	defConfName = "server.toml"
)

var conf *Conf = &defaultConfig

func listen(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		log.Printf("New connection from %s", conn.RemoteAddr())
		go (&Client{rwc: conn}).Handle()
	}
}

func main() {
	var (
		confFile = flag.String("conf", defConfName, "Name of configuration file")
	)

	flag.UintVar(&conf.TCP.Port, "port", 2671, "Port for TCP connections")
	flag.UintVar(&conf.Web.Port, "webport", 8080, "Port for HTTP connections")
	flag.BoolVar(&conf.WS.Enabled, "websocket", false, "Listen for websocket upgrades only")
	flag.StringVar(&conf.Database, "db", "kalah.sql", "Path to SQLite database")
	flag.UintVar(&conf.Game.Timeout, "timeout", 5, "Seconds to wait for a move to be made")
	flag.BoolVar(&conf.Debug, "debug", false, "Print all network I/O")
	flag.Parse()

	newconf, err := openConf(*confFile)
	if err != nil && (!os.IsNotExist(err) || *confFile != defConfName) {
		log.Fatal(err)
	}
	if newconf != nil {
		conf = newconf
	}

	if conf.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if conf.WS.Enabled {
		http.HandleFunc("/socket", listenUpgrade)
		log.Println("Listening for upgrades on /socket")
	} else {
		tcp := fmt.Sprintf("%s:%d", conf.TCP.Host, conf.TCP.Port)
		plain, err := net.Listen("tcp", tcp)
		if err != nil {
			log.Fatal(err)
		}
		go listen(plain)
	}

	// Start web server
	go func() {
		web := fmt.Sprintf("%s:%d", conf.Web.Host, conf.Web.Port)
		log.Fatal(http.ListenAndServe(web, nil))
	}()

	// start match scheduler
	go organizer()

	// start database manager
	manageDatabase(conf.Database)
}
