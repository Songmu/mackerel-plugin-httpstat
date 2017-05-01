package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tcnksm/go-httpstat"
)

const (
	exitOK = iota
	exitError
)

func main() {
	os.Exit(run(os.Args[1:]))
}

var sanitizeReg = regexp.MustCompile(`[^A-Za-z0-9_-]`)

func run(argv []string) int {
	fs := flag.NewFlagSet("mackerel-plugin-httpstat", flag.ContinueOnError)
	var (
		targetURL  string
		httpMethod string
		httpBody   string
		metricKey  string
	)
	fs.StringVar(&targetURL, "url", "", "target URL")
	fs.StringVar(&httpMethod, "method", "GET", "http method")
	fs.StringVar(&httpBody, "body", "", "request body(optional)")
	fs.StringVar(&metricKey, "metric-key", "", "metric key (generated from url by default)")
	if err := fs.Parse(argv); err != nil {
		return exitError
	}

	var body io.Reader
	if httpBody != "" {
		body = strings.NewReader("")
	}
	req, err := http.NewRequest(httpMethod, targetURL, body)
	if err != nil {
		log.Println(err)
		return exitError
	}
	statRes := &httpstat.Result{}
	ctx := httpstat.WithHTTPStat(req.Context(), statRes)
	req = req.WithContext(ctx)

	startAt := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return exitError
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	endAt := time.Now()
	statRes.End(endAt)

	if metricKey == "" {
		metricKey = strings.Replace(targetURL, "://", "_", 1)
	}
	metricKey = sanitizeReg.ReplaceAllString(metricKey, "_")

	precision := time.Millisecond
	keyfmt := "httpstat." + metricKey + ".%s"
	linefmt := "%s\t%d\t" + fmt.Sprintf("%d\n", startAt.Unix())

	fmt.Printf(linefmt, fmt.Sprintf(keyfmt, "dnslookup"), statRes.DNSLookup/precision)
	fmt.Printf(linefmt, fmt.Sprintf(keyfmt, "tcpconnection"), statRes.TCPConnection/precision)
	fmt.Printf(linefmt, fmt.Sprintf(keyfmt, "tlshandshake"), statRes.TLSHandshake/precision)
	fmt.Printf(linefmt, fmt.Sprintf(keyfmt, "serverprocessing"), statRes.ServerProcessing/precision)
	fmt.Printf(linefmt, fmt.Sprintf(keyfmt, "contenttransfer"), statRes.ContentTransfer(endAt)/precision)

	return exitOK
}
