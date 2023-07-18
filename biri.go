package biri

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type config struct {
	PingServer             string
	proxyWebpage           string
	TickMinuteDuration     time.Duration
	numberAvailableProxies int
	Verbose                int
}

// Config configuration
var Config = &config{
	proxyWebpage:           "https://free-proxy-list.net/",
	PingServer:             "https://www.google.com/",
	TickMinuteDuration:     3,
	numberAvailableProxies: 30,
	Verbose:                1,
}

// SkipProxies contains not working proxies rip
var SkipProxies = []string{}

var availableProxies = make(chan *Proxy, Config.numberAvailableProxies)
var reAddedProxies = []Proxy{}
var banProxy = make(chan string)
var done = make(chan bool)
var timeout = 10

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
	log.Println("Ban", p.Info)
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

func getProxy() {
	if Config.Verbose > 0 {
		log.Println("Get new proxies")
	}
	_, cancel := context.WithTimeout(context.Background(), time.Duration(timeout))
	defer cancel()

	response, errGet := http.Get(Config.proxyWebpage)
	if errGet != nil {
		log.Println("Error on get proxy")
		return
	}
	defer response.Body.Close()

	query, errParse := goquery.NewDocumentFromReader(response.Body)
	if errParse != nil {
		return
	}

	query.Find("table tr").Each(func(_ int, proxyLi *goquery.Selection) {
		if strings.Contains(proxyLi.Text(), "elite proxy") {
			if proxyLi.Children().Filter("td.hx").Text() == "yes" {

				ip := proxyLi.Children().First()
				res := fmt.Sprintf("%v:%v", ip.Text(), ip.Next().Text())

				for _, val := range SkipProxies {
					if res == val {
						return
					}
				}
				go basicTestProxy(res)
			}
		}
	})
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
		Timeout: time.Duration(timeout) * time.Second,
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
