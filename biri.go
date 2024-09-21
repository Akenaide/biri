package biri

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type config struct {
	// HttpProxyListGenerator generates a list of proxies to use.
	// The returned strings must not have "http://" prepended.
	HttpProxyListGenerator func() ([]string, error)
	PingServer             string
	TickMinuteDuration     time.Duration
	numberAvailableProxies int
	Verbose                int
	Timeout                int
}

// Config configuration
var Config = &config{
	HttpProxyListGenerator: FreeProxyListExtractor,
	PingServer:             "https://www.google.com/",
	TickMinuteDuration:     3,
	numberAvailableProxies: 30,
	Verbose:                1,
	Timeout:                10,
}

// SkipProxies contains not working proxies rip
var SkipProxies = []string{}

var availableProxies = make(chan *Proxy, Config.numberAvailableProxies)
var reAddedProxies = []Proxy{}
var banProxy = make(chan string)
var done = make(chan bool)

func removeWithIndex(s []Proxy, index int) []Proxy {
	ret := []Proxy{}
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// Proxy handle proxy things
type Proxy struct {
	Info   string
	Client *http.Client
}

// Readd good proxy
func (p *Proxy) Readd() {
	reAddedProxies = append(reAddedProxies, *p)
	go func() {
		availableProxies <- p
	}()
}

// Ban proxy
func (p *Proxy) Ban() {
	slog.Debug(fmt.Sprintf("Ban %v", p.Info))
	toBanIndex := -1
	for index, proxy := range reAddedProxies {
		if p.Info == proxy.Info {
			log.Println("We found a banned proxy in reAddedProxies at:", index)
			toBanIndex = index
			break
		}

	}
	if toBanIndex != -1 {
		reAddedProxies = removeWithIndex(reAddedProxies, toBanIndex)
	}
	banProxy <- p.Info
}

// ProxyStart start channels
func ProxyStart() {
	ticker := time.NewTicker(Config.TickMinuteDuration * time.Minute)
	go getProxy()

	go func() {
		for {
			select {
			case skip := <-banProxy:
				SkipProxies = append(SkipProxies, skip)
			case <-ticker.C:
				go getProxy()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}

// GetClient return client with proxy
func GetClient() *Proxy {
	select {
	case proxy := <-availableProxies:
		return proxy
	default:
		if len(reAddedProxies) > 0 {
			randomIndex := rand.Intn(len(reAddedProxies))
			return &reAddedProxies[randomIndex]
		} else {
			return <-availableProxies
		}
	}
}

// FreeProxyListExtractor gets proxies from https://free-proxy-list.net/.
func FreeProxyListExtractor() ([]string, error) {
	response, err := http.Get("https://free-proxy-list.net/")
	if err != nil {
		log.Println("Error on get proxy")
		return nil, fmt.Errorf("error getting website: %v", err)
	}
	defer response.Body.Close()

	query, errParse := goquery.NewDocumentFromReader(response.Body)
	if errParse != nil {
		return nil, fmt.Errorf("error parsing response: %v", errParse)
	}

	anonymousLevel := []string{"elite proxy", "transparent"}
	var proxies []string
	query.Find("table tr").Each(func(_ int, proxyLi *goquery.Selection) {
		for _, anoLevel := range anonymousLevel {
			if strings.Contains(proxyLi.Text(), anoLevel) {
				if proxyLi.Children().Filter("td.hx").Text() == "yes" {

					ip := proxyLi.Children().First()
					res := fmt.Sprintf("%v:%v", ip.Text(), ip.Next().Text())

					for _, val := range SkipProxies {
						if res == val {
							return
						}
					}
					proxies = append(proxies, res)
					// Return so we don't add the same proxy twice.
					return
				}
			}
		}
	})

	slog.Info(fmt.Sprintf("Got %v new proxies", len(proxies)))

	return proxies, nil
}

// ProxyScraperListExtractor gets proxies from https://github.com/ProxyScraper/ProxyScraper/tree/main's http list.
func ProxyScraperListExtractor() ([]string, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/refs/heads/main/http.txt")
	if err != nil {
		return nil, fmt.Errorf("error getting proxy list: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing proxy list: %v", err)
	}

	proxyRE, err := regexp.Compile(`(\d+\.\d+\.\d+\.\d+:\d+)`)
	if err != nil {
		return nil, fmt.Errorf("issue with the proxy regex: %v", err)
	}
	matches := proxyRE.FindAllString(doc.Text(), -1)

	slog.Info(fmt.Sprintf("Got %d proxies", len(matches)))
	return matches, nil
}

func getProxy() {
	_, cancel := context.WithTimeout(context.Background(), time.Duration(Config.Timeout))
	defer cancel()

	proxies, err := Config.HttpProxyListGenerator()
	if err != nil {
		log.Printf("Error getting proxies: %v\n", err)
		return
	}

proxyLoop:
	for _, p := range proxies {
		for _, val := range SkipProxies {
			if p == val {
				continue proxyLoop
			}
		}
		go basicTestProxy(p)
	}
}

func basicTestProxy(p string) {
	proxy := Proxy{Info: p}
	proxyURL, err := url.Parse(fmt.Sprintf("http://%v", proxy.Info))
	if err != nil {
		log.Println("Error in parse url")
	}
	proxy.Client = &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	},
		Timeout: time.Duration(Config.Timeout) * time.Second,
	}

	_, errHTTP := proxy.Client.Get(Config.PingServer)
	if errHTTP != nil {
		proxy.Ban()
		return
	}
	if Config.Verbose > 2 {
		log.Println("Good proxy", proxy.Info)
	}
	availableProxies <- &proxy
}

// Done stop ticker and channels
func Done() {
	done <- true
}
