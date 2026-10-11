package main

// Built-in traffic generator to populate Avi analytics:
//   erpsock loadgen [-c 10] [-d 60s] [-k] [-attacks] http://vip.example
// Each worker keeps its own cookie jar, like a browser.

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type weighted struct {
	weight int
	path   func() string
}

func fixed(p string) func() string { return func() string { return p } }

var traffic = []weighted{
	{20, fixed("/")},
	{15, fixed("/info")},
	{10, fixed("/sales")},
	{10, fixed("/stock")},
	{10, fixed("/accounting")},
	{8, fixed("/api/sales")},
	{8, fixed("/api/stock")},
	{5, fixed("/api/accounting?limit=500")},
	{5, fixed("/session")},
	{4, func() string { return fmt.Sprintf("/slow?ms=%d", 100+rand.Intn(1400)) }},
	{3, func() string { return "/error?code=" + []string{"404", "500", "503"}[rand.Intn(3)] }},
	{2, fixed("/cpu?ms=50")},
}

var attacks = []string{
	"' OR '1'='1",
	"1; DROP TABLE customers--",
	"<script>alert(document.cookie)</script>",
	"../../../../etc/passwd",
	"\" onmouseover=\"alert(1)",
}

func loadgen(args []string) {
	fs := flag.NewFlagSet("loadgen", flag.ExitOnError)
	conc := fs.Int("c", 10, "concurrent workers")
	dur := fs.Duration("d", 60*time.Second, "test duration (0 = until interrupted)")
	insecure := fs.Bool("k", false, "skip TLS certificate verification")
	withAttacks := fs.Bool("attacks", false, "add SQLi/XSS/traversal requests (WAF demo)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: erpsock loadgen [flags] http://vip")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	base := strings.TrimRight(fs.Arg(0), "/")

	mix := traffic
	if *withAttacks {
		mix = append(mix[:len(mix):len(mix)], weighted{10, func() string {
			return "/search?q=" + url.QueryEscape(attacks[rand.Intn(len(attacks))])
		}})
	}
	total := 0
	for _, w := range mix {
		total += w.weight
	}
	next := func() string {
		n := rand.Intn(total)
		for _, w := range mix {
			if n < w.weight {
				return w.path()
			}
			n -= w.weight
		}
		return "/"
	}

	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: *insecure},
		MaxIdleConnsPerHost: *conc,
	}
	var (
		reqs, errs, latencyUs atomic.Int64
		classes               [6]atomic.Int64
		wg                    sync.WaitGroup
		stop                  = make(chan struct{})
	)
	for i := 0; i < *conc; i++ {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Transport: transport, Jar: jar, Timeout: 70 * time.Second}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				start := time.Now()
				resp, err := client.Get(base + next())
				if err != nil {
					if errs.Add(1) == 1 {
						log.Println("first error:", err)
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				reqs.Add(1)
				latencyUs.Add(time.Since(start).Microseconds())
				classes[clamp(resp.StatusCode/100, 0, 5)].Add(1)
			}
		}()
	}

	log.Printf("loadgen: %d workers -> %s for %v", *conc, base, *dur)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var deadline <-chan time.Time
	if *dur > 0 {
		deadline = time.After(*dur)
	}
	report := func() {
		n := reqs.Load()
		avg := 0.0
		if n > 0 {
			avg = float64(latencyUs.Load()) / float64(n) / 1000
		}
		log.Printf("requests=%d 2xx=%d 3xx=%d 4xx=%d 5xx=%d errors=%d avg=%.1fms",
			n, classes[2].Load(), classes[3].Load(), classes[4].Load(), classes[5].Load(), errs.Load(), avg)
	}
	for {
		select {
		case <-ticker.C:
			report()
		case <-deadline:
			close(stop)
			wg.Wait()
			report()
			return
		}
	}
}
