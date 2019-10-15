package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
)

var relative_links_regex = regexp.MustCompile(`href="(?P<link>\S+)"`)
var explored_links []string // pour ne pas tourner en rond il faut savoir d'où on vient

var run = true // permet d'arrêter l'itération des boucles & la récursion

func main() {
	unix_chan := make(chan os.Signal, 1)   // merci stackoverflow
	signal.Notify(unix_chan, os.Interrupt) // sert à afficher les liens trouvés avant de fermer avec CTRL+C, pour les gens pressés
	go func() {
		for _ = range unix_chan {
			run = false
			fmt.Println()
			fmt.Println("       * -- STOP -- *")
		}
	}()

	var all_insecure_links []string
	var all_secure_links []string
	til, tis := recurse("https://www.insa-lyon.fr/", "https://www.insa-lyon.fr", all_insecure_links, all_secure_links, 2)

	fmt.Println("-------------------------")
	fmt.Println("Fin de l'indexation")
	for _, l := range tis {
		fmt.Println(l)
	}
	for _, l := range til {
		fmt.Println(l)
	}
}

func stringInStrings(string string, strings []string) bool {
	// exactement la même utilité que le 'in' de python
	for _, s := range strings {
		if s == string {
			return true
		}
	}
	return false
}

// link le lien à parcourir
// root l'adresse du site, pour évider d'indexer tout le WorldWide Web
// n le nombre maximal de récursions
func recurse(link string, root string, total_insecure_links []string, total_secure_links []string, n int) (insecure_links []string, secure_links []string) {
	insecure_links_page, secure_links_page := crawl(link, root)
	explored_links = append(explored_links, link)
	total_insecure_links = append(total_insecure_links, insecure_links_page...) // on enregistre les liens http

	for _, link := range secure_links_page { // on parcourt tous les liens HTTPS pour trouver les liens http
		total_secure_links = append(total_secure_links, link)
		if strings.Contains(link, root) && run {
			// si on reste dans le même site
			if !stringInStrings(link, explored_links) && n > 0 { // si on a pas déjà visité le lien et qu'on a le droit de boucler
				recurse(link, root, total_insecure_links, total_secure_links, n-1) // on continue
			}
		}
	}
	return total_insecure_links, total_secure_links
}

func extract_hrefs(page []byte, current_page string, root string) (http_links []string, https_links []string) {
	mat := relative_links_regex.FindAllSubmatch(page, -1)
	for _, match := range mat { // le premier élement du match est le submatch (il ne peut y en a voir qu'un seul)
		link := string(match[1])
		//fmt.Print(link)
		if link[0] == '#' { // c'est un lien-ancre, il va rester sur la même page
			continue
		}
		if strings.Contains(link, "https://") {
			//fmt.Println(" s")
			https_links = append(https_links, link)
			continue
		}
		if strings.Contains(link, "http://") {
			//fmt.Println(" is")
			http_links = append(http_links, link)
			continue
		}
		if link[0] == '?' { //le lien diffère juste des query params url, donc il faut le reconstruire
			// pas sûr que ça permette de trouver plus d'infos
			//fmt.Println("bf r "+link)
			link = current_page + link
			//fmt.Println(" -> ref to " + link)
			https_links = append(https_links, link)
			continue
		}
		if link[0] == '/' { //le lien est relatif (/index.html), donc il faut le reconstruire
			link = root + link
			//fmt.Println(" -> ref to " + link)
			https_links = append(https_links, link)
			continue
		} else {
			_, _ = fmt.Print("[err: " + link + "] ")
		}
	}
	return http_links, https_links
}

func crawl(site string, root string) (http_links []string, https_links []string) {
	fmt.Print("Exploring page " + site + " for links")

	page, err := http.Get(site)
	defer page.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	corpsPage, err := ioutil.ReadAll(page.Body)

	liensHttp, liensHttps := extract_hrefs(corpsPage, site, root)
	fmt.Println(" ... " + strconv.Itoa(len(liensHttps)) + " secure found")

	return liensHttp, liensHttps
}
