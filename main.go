package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/bobappleyard/readline"
	"github.com/codegangsta/cli"

	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func clear_prompt() {
	fmt.Printf("\033[2K\033[E")
}

func print_prompt() {
	fmt.Printf("> ")
}

var colors = map[string]string{
	"red":     "\033[31m",
	"green":   "\033[32m",
	"yellow":  "\033[33m",
	"blue":    "\033[34m",
	"default": "\033[39m",
}

func print(msg string, color string) {
	c := colors[color]
	fmt.Printf(c + msg + colors["default"])
}

func readWS(c chan []byte, q chan int, ws *websocket.Conn) {
	for {
		msg := make([]byte, 512)
		n, err := ws.Read(msg)
		if err != nil {
			break
		}
		c <- msg[:n]
	}
	q <- 0
}

func readLine(c chan []byte) {
	for {
		line, err := readline.String("> ")
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		readline.AddHistory(line)
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, "\r")
		msg := []byte(line)
		c <- msg
	}
}

func console(ws *websocket.Conn) {
	ch1 := make(chan []byte)
	q := make(chan int)
	ch2 := make(chan []byte)
	go readWS(ch1, q, ws)
	go readLine(ch2)

	for {
		select {
		case <-q:
			return
		case msg := <-ch1:
			clear_prompt()
			print("< "+string(msg)+"\n", "blue")
			print_prompt()
		case msg := <-ch2:
			_, err := ws.Write(msg)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

var connection_count int
var connection_mutex sync.Mutex

func get_connection() bool {
	connection_mutex.Lock()
	defer connection_mutex.Unlock()
	if connection_count == 0 {
		connection_count++
		return true
	} else {
		return false
	}
}

func return_connection() {
	connection_mutex.Lock()
	defer connection_mutex.Unlock()
	connection_count--
}

func echoHandler(ws *websocket.Conn) {
	if get_connection() {
		defer func() {
			return_connection()
			clear_prompt()
			print("disconnected\n", "green")
		}()
		clear_prompt()
		print("client connected\n", "green")
		console(ws)
	}
}

func client(ws *websocket.Conn) {
	defer func() {
		clear_prompt()
		print("disconnected\n", "green")
	}()
	print("connected (press CTRL+C to quit)\n", "green")

	console(ws)
}

func listen(c *cli.Context) {
	if len(c.Args()) == 0 {
		log.Fatal("specify port: wsgat listen <port>")
	}

	port := c.Args()[0]
	addr := ":" + port

	http.Handle("/", websocket.Handler(echoHandler))
	print("listening on port "+port+" (press CTRL+C to quit)\n", "green")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func connect(c *cli.Context) {
	if len(c.Args()) == 0 {
		log.Fatal("specify url: wsgat connect <url>")
	}

	url := c.Args()[0]
	protocol := c.Int("protocol")
	origin := c.String("origin")
	subprotocol := c.String("subprotocol")
	auth := c.String("auth")
	header := c.StringSlice("header")

	config, err := websocket.NewConfig(url, origin)
	if err != nil {
		log.Fatal(err)
	}
	if subprotocol != "" {
		config.Protocol = []string{subprotocol}
	}
	config.Version = protocol

	httpHeader := make(http.Header)
	if auth != "" {
		httpHeader.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	for _, h := range header {
		s := strings.Split(h, ":")
		if len(s) != 2 {
			log.Fatal("invalid header: " + h)
		}
		httpHeader.Add(s[0], s[1])
	}

	config.Header = httpHeader
	ws, err := websocket.DialConfig(config)

	if err != nil {
		log.Fatal(err)
	}

	client(ws)
}

func main() {
	lFlags := []cli.Flag{}

	cFlags := []cli.Flag{
		cli.IntFlag{
			Name:  "protocol, p",
			Value: 13,
			Usage: "protocol version",
		},
		cli.StringFlag{
			Name:  "origin, o",
			Value: "http://localhost/",
			Usage: "origin",
		},
		cli.StringFlag{
			Name:  "subprotocol, s",
			Value: "",
			Usage: "subprotocol",
		},
		cli.StringFlag{
			Name:  "auth",
			Value: "",
			Usage: "Add basic HTTP authentication header.",
		},
		cli.StringSliceFlag{
			Name:  "header, H",
			Value: &cli.StringSlice{},
			Usage: "Set an HTTP header. Repeat to set multiple.",
		},
	}

	app := cli.NewApp()
	app.Name = "wsgat"
	app.Usage = `
   wsgat listen <port> [options]
   wsgat connect <url> [options]
  `
	app.Commands = []cli.Command{
		{
			Name:   "listen",
			Usage:  "listen on port",
			Flags:  lFlags,
			Action: listen,
		},
		{
			Name:   "connect",
			Usage:  "connect to a websocket server",
			Flags:  cFlags,
			Action: connect,
		},
	}

	app.Run(os.Args)
}
