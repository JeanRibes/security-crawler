package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	//"net/url"
)

func main() {
	var all_html_links []string
	recurse(("https://vps.ribes.ovh/"), all_html_links)
}
func recurse(olink string, total_http_list []string) (links []string) {
	links_http, https_links := crawl(olink)
	total_http_list = append(total_http_list, links_http...)
	for _, link := range https_links { // on parcourt tous les liens HTTPS pour trouver les liens http
		if strings.Contains(olink, link) { // c'est pour rester dans le même site TODO: utiliser une bonne regex
			total_http_list = append(total_http_list, recurse(link, total_http_list)...)
		}
	}
	return total_http_list
}

//func crawl(site url.URL) ([]url.URL) {
func crawl(site string) (http_links []string, https_links []string) {

	http_regex := regexp.MustCompile(`http://(\w|\.)*\.\w+`)
	https_regex := regexp.MustCompile(`https://(\w|\.)*\.\w+`)
	//http_regex := regexp.MustCompile(`http://.*`)
	//https_regex := regexp.MustCompile(`https://.*`)
	page, err := http.Get(site)
	if err != nil {
		fmt.Println(err)
	}
	defer page.Body.Close()
	corpsPage, err := ioutil.ReadAll(page.Body)
	matchs_http := http_regex.FindAll(corpsPage, -1)
	matchs_https := https_regex.FindAll(corpsPage, -1)
	var liensHttp []string
	var liensHttps []string
	for _, match := range matchs_http {
		liensHttp = append(liensHttp, string(match))
		fmt.Println(string(match))
	}
	for _, match := range matchs_https {
		liensHttps = append(liensHttps, string(match))
		fmt.Println(string(match))
	}
	return liensHttp, liensHttps
}
