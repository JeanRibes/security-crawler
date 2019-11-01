package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
)

var relative_links_regex = regexp.MustCompile(`href="(?P<link>\S+)"`)
var explored_links []string // pour ne pas tourner en rond il faut savoir d'où on vient
var explored_links_lock sync.RWMutex

var all_insecure_links []string
var all_secure_links []string
var total_links_list_lock sync.RWMutex

var run = true // permet d'arrêter l'itération des boucles & la récursion
var wg sync.WaitGroup

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

	to_index := "https://www.insa-lyon.fr"
	to_index = "https://jean.ribes.ovh"
	if len(os.Args) == 2 {
		to_index = os.Args[1]
	}
	crawing_loop(to_index, to_index)

	fmt.Println("-------------------------")
	fmt.Println("Fin de l'indexation")
	/*for _, l := range all_insecure_links {
		fmt.Println(l)
	}
	for _, l := range all_secure_links {
		fmt.Println(l)
	}*/
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

func crawing_loop(page string, root string) {
	to_explore := &[]string{root}
	for len(*to_explore) > 0 {
		next_loop_explore := &[]string{}
		explored_links = append(explored_links, *to_explore...) // en fait on pourait nettoyer dans chaque goroutine
		for _, link := range *to_explore {                      // on lance toutes les requetes en même temps
			wg.Add(1)
			go poolcrawl(link, root, next_loop_explore)
		}
		wg.Wait()
		fmt.Println("batch finie")
		fmt.Println(len(*next_loop_explore))
		deduplicateur(*next_loop_explore)
		fmt.Println(len(*next_loop_explore))
		to_explore = next_loop_explore // on échange les variables
	}
}

var to_explore_lock sync.RWMutex

func poolcrawl(page string, root string, to_explore *[]string) {
	defer wg.Done()
	if strings.Contains(page, root) && run {
		insecure_links_page, secure_links_page := crawl(page, root)
		total_links_list_lock.Lock()
		all_insecure_links = append(all_insecure_links, insecure_links_page...) // on enregistre les liens http
		all_secure_links = append(all_secure_links, secure_links_page...)
		total_links_list_lock.Unlock()
		/*fmt.Println(len(explored_links))
		fmt.Println(len(secure_links_page))
		fmt.Println(len(*to_explore))*/

		var local_to_add []string
		for _, link := range secure_links_page {
			if strings.Contains(link, root) {
				local_to_add = append(local_to_add, link)
			}
		}
		to_explore_lock.Lock()
		*to_explore = append(*to_explore, local_to_add...) // y'aura des doublons mais on nettoie après
		//fmt.Println(len(*to_explore))
		to_explore_lock.Unlock() //
	}
}
func deduplicateur(multiples []string) []string {
	var uniques []string
	for _, elem := range multiples {
		unique := true
		for _, uniq := range uniques { // si elem n'est pas dans uniques
			if uniq == elem { //si on le trouve
				unique = false
			}
		}
		if unique {
			uniques = append(uniques, elem)
		}
	}
	return uniques
}
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
			//_, _ = fmt.Print("[err: " + link + "] ")
		}
	}
	return http_links, https_links
}

func crawl(site string, root string) (http_links []string, https_links []string) {
	//fmt.Print("Exploring page " + site + " for links")

	page, err := http.Get(site)
	if err != nil {
		fmt.Println(err)
		return []string{}, []string{}
	}
	if strings.Contains(page.Header.Get("content-type"), "text/html") {
		corpsPage, rerr := ioutil.ReadAll(page.Body)
		if rerr == nil {
			liensHttp, liensHttps := extract_hrefs(corpsPage, site, root)
			//fmt.Println(" ... " + strconv.Itoa(len(liensHttps)) + " secure found")
			err2 := page.Body.Close()
			if err2 != nil {
				fmt.Println(err)
			} else {
				return liensHttp, liensHttps
			}
		} else {
			fmt.Println(rerr)
		}
	}
	return []string{}, []string{}
}
