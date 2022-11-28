package p2g

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"
)

var mutex = &sync.Mutex{}
var proxyRegex = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
var proxyUrls = []string{
	"https://api.proxyscrape.com/v2/?request=getproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=all",
	"https://raw.githubusercontent.com/mertguvencli/http-proxy-list/main/proxy-list/data.txt",
	"https://raw.githubusercontent.com/saschazesiger/Free-Proxies/master/proxies/http.txt",
	"https://github.com/jetkai/proxy-list/blob/main/online-proxies/txt/proxies-https.txt",
	"https://github.com/jetkai/proxy-list/blob/main/online-proxies/txt/proxies-http.txt",
	"https://github.com/BlackSnowDot/proxylist-update-every-minute/blob/main/https.txt",
	"https://github.com/BlackSnowDot/proxylist-update-every-minute/blob/main/http.txt",
	"https://raw.githubusercontent.com/UptimerBot/proxy-list/main/proxies/http.txt",
	"https://github.com/roosterkid/openproxylist/blob/main/HTTPS_RAW.txt",
	"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt",
	"https://raw.githubusercontent.com/proxy4parsing/proxy-list/main/http.txt",
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
	"https://raw.githubusercontent.com/hyperbeats/proxy-list/main/http.txt",
	"https://raw.githubusercontent.com/mmpx12/proxy-list/master/http.txt",
	"https://sunny9577.github.io/proxy-scraper/proxies.txt",
	"https://www.proxy-list.download/api/v1/get?type=https",
	"https://www.proxy-list.download/api/v1/get?type=http",
	"https://www.proxyscan.io/download?type=https",
	"https://www.proxyscan.io/download?type=http",
	"https://api.openproxylist.xyz/http.txt",
	"http://pubproxy.com/api/proxy?limit=5",
	"https://openproxy.space/list/http",
	"https://www.juproxy.com/free_api",
}

type P2G struct {
	allProxies     map[string]int
	currentProxies map[string]int
	ProxyCount     int
}

func requestProxies(url string) string {
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return string(body)
}

func getProxiesFromUrl(url string) []string {
	proxyList := requestProxies(url)
	proxies := proxyRegex.FindAllString(proxyList, -1)
	return proxies
}

func getProxies() []string {
	var proxies []string
	var wg sync.WaitGroup
	for _, url := range proxyUrls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			proxySlice := getProxiesFromUrl(url)

			mutex.Lock()
			defer mutex.Unlock()
			proxies = append(proxies, proxySlice...)
		}(url)
	}
	wg.Wait()

	return proxies
}

func deDupe(proxies []string) []string {
	keyMap := make(map[string]bool)
	for _, proxy := range proxies {
		keyMap[proxy] = true
	}

	var result []string
	for proxy := range keyMap {
		result = append(result, proxy)
	}

	return result
}

func (p2g *P2G) GetProxyList() []string {
	proxies := getProxies()
	proxies = deDupe(proxies)

	return proxies
}

func (p2g *P2G) getUseableProxy() string {
	mutex.Lock()
	defer mutex.Unlock()
	if len(p2g.currentProxies) == 0 {
		if len(p2g.allProxies) == 0 {
			fmt.Println("No proxies found, getting proxies...")
			p2g.allProxies = make(map[string]int)
			for _, proxy := range p2g.GetProxyList() {
				p2g.allProxies[proxy] = 0
			}
		} else {
			fmt.Println("No proxies left, refilling...")
		}
		for proxy := range p2g.allProxies {
			p2g.currentProxies[proxy] = 0
		}
	}

	for proxy := range p2g.currentProxies {
		delete(p2g.currentProxies, proxy)
		return proxy
	}

	return ""
}

func (p2g *P2G) markBadProxy(proxy string) {
	mutex.Lock()
	defer mutex.Unlock()
	if p2g.allProxies[proxy] > 3 {
		delete(p2g.allProxies, proxy)
	} else {
		p2g.allProxies[proxy]++
	}
}

func (p2g *P2G) Get(requestUrl string) (*http.Response, error) {
	proxy := p2g.getUseableProxy()
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   proxy,
			}),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(requestUrl)
	if err != nil {
		p2g.markBadProxy(proxy)
		return nil, err
	}

	return resp, err
}

func (p2g *P2G) SetupProxies() {
	proxies := p2g.GetProxyList()

	p2g.ProxyCount = len(proxies)
	p2g.allProxies = make(map[string]int)
	p2g.currentProxies = make(map[string]int)

	for _, proxy := range proxies {
		p2g.allProxies[proxy] = 0
		p2g.currentProxies[proxy] = 0
	}
}

func NewP2G() *P2G {
	p2g := &P2G{}
	return p2g
}
