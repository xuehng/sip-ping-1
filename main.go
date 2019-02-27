package main

import (
	"bufio"
	"bytes"

	"flag"
	"fmt"
	"log"
	mathrand "math/rand"
	"net"
	"net/url"
	"os"

	"strings"

	"text/template"
	"time"
)

type SipOption struct {
	Url, Localhost, Localport string
	Callid, Cseq, Branchrand  int
}

var addr = flag.String("addr", "", "http service address")
var skipVerify = flag.Bool("skipverify", false, "skip TLS certificate verification")

const UDP_OPTIONS = `OPTIONS {{.Url}};transport=udp SIP/2.0
Call-ID: {{.Callid}}@{{.Localhost}}
CSeq: {{.Cseq}} OPTIONS
Max-Forwards: 70
From: SIP-PING <sip:sip-ping@invalid>;tag=00-01991-00116239-25a473e47
To: <sip:host@invalid;transport=udp>
Via: SIP/2.0/UDP {{.Localhost}}:{{.Localport}};branch=z9hG4bK{{.Branchrand}}
Accept: application/sdp
Content-Length: 0

` // two newlines required to signal end of request

func renderRequest(options string, url string, localaddr string) *bytes.Buffer {
	localAddrInfos := strings.Split(localaddr, ":")
	if len(localAddrInfos) < 2 {
		return nil
	}

	mathrand.Seed(time.Now().UnixNano())

	data := SipOption{
		Url:        url,
		Localhost:  localAddrInfos[0],
		Localport:  localAddrInfos[1],
		Callid:     mathrand.Intn(1<<31 - 1),
		Cseq:       mathrand.Intn(1<<31 - 1),
		Branchrand: mathrand.Intn(1<<31 - 1),
	}
	tmp, err := template.New("name1").Parse(options)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	var buf bytes.Buffer
	if err = tmp.Execute(&buf, data); err != nil {
		log.Println(err.Error())
		return nil
	}
	return &buf
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	if len(*addr) == 0 {
		flag.Usage()
		os.Exit(255)
	}

	var url, err = url.Parse(*addr)
	if err != nil {
		log.Fatal("addr:", err)
	}

	if url.Scheme == "sip" {
		conn, err := net.DialTimeout("udp", url.Host, (time.Duration(5) * time.Second))
		if err != nil {
			log.Fatal("dial:", err)
			return
		}
		defer conn.Close()

		buf := renderRequest(UDP_OPTIONS, *addr, conn.LocalAddr().String())
		if buf == nil {
			log.Println("render request fail")
			return
		}

		fmt.Println(buf)

		conn.SetDeadline(time.Now().Add(time.Duration(5) * time.Second))

		if _, err = conn.Write(buf.Bytes()); err != nil {
			log.Println("send request fail")
			return
		}

		reader := bufio.NewReader(conn)
		scanner := bufio.NewScanner(reader)
		scanner.Split(bufio.ScanLines)

		var ok200 bool

		for noErr := scanner.Scan(); noErr == true && scanner.Text() != ""; scanner.Scan() {
			fmt.Println(scanner.Text()) // scanner.Bytes()

			if strings.Contains(scanner.Text(), "SIP/2.0 200 OK") {
				ok200 = true
			}
		}

		fmt.Print("reachable: ", ok200)
	} else {
		log.Println("Unknown scheme:", url.Scheme)
		os.Exit(255)
	}
}
