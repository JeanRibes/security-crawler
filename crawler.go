package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var relative_links_regex = regexp.MustCompile(`href="(?P<link>\S+)"`)

var run = true // permet d'arrêter l'itération des boucles & la récursion

func main() {
	ln, err := net.Listen("tcp", ":1984")
	if err != nil {
		// handle error
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
		}
		go handleClient(conn)
	}
	to_index := "https://www.insa-lyon.fr"
	to_index = "https://jean.ribes.ovh"
	if len(os.Args) == 2 {
		to_index = os.Args[1]
	}
	//all_insecure_links, all_secure_links := crawing_loop(to_index, to_index)
	all_insecure_links, _ := crawing_loop(to_index, to_index)

	fmt.Println("-------------------------")
	fmt.Println("Fin de l'indexation")
	for _, l := range all_insecure_links {
		fmt.Println(l)
	}
	/*for _, l := range all_secure_links {
		fmt.Println(l)
	}*/
}

func handleClient(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	//writer.WriteString("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04²")
	//writer.WriteString("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04")
	//writer.Write([]byte("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04"))
	sendString(writer, "Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)")

	//lien, err := reader.ReadString('\x04')
	lien, err := recvString(reader)
	if err != nil {
		print(err)
	}
	//writer.WriteString(strconv.Itoa(to_index(strings.TrimSuffix(lien, "²"))) + "\x04²")
	//writer.Flush()
	sendString(writer, "Pourcentage de liens non sécurisés : "+strconv.Itoa(to_index(lien))+"%")
}
func to_index(website string) int {
	insecure, secure := crawing_loop(website, website)
	return int((100 * float64(len(insecure))) / float64(len(secure)))
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

func crawing_loop(page string, root string) ([]string, []string) {
	wg := &sync.WaitGroup{}
	explored_linksP := &[]string{}
	//explored_links_lock := &sync.RWMutex{}
	total_links_list_lockP := &sync.RWMutex{}
	all_insecure_linksP := &[]string{}
	all_secure_linksP := &[]string{}
	to_explore := &[]string{root}
	for len(*to_explore) > 0 {
		next_loop_explore := &[]string{}
		*explored_linksP = append(*explored_linksP, *to_explore...) // en fait on pourait nettoyer dans chaque goroutine
		for _, link := range *to_explore {                          // on lance toutes les requetes en même temps
			wg.Add(1)
			go poolcrawl(link, root, next_loop_explore, explored_linksP, all_insecure_linksP, all_secure_linksP, total_links_list_lockP, wg)
		}
		wg.Wait()
		fmt.Println("batch finie, dédupication des URL")
		*all_secure_linksP = deduplicateur(*all_secure_linksP)
		*all_insecure_linksP = deduplicateur(*all_insecure_linksP)
		fmt.Println(len(*next_loop_explore))
		fmt.Println("nouvelle batch")
		*next_loop_explore = deduplicateur(*next_loop_explore)
		fmt.Println(len(*next_loop_explore))
		to_explore = next_loop_explore // on échange les variables
	}
	/*print("deddup finale")
	deduplicateur(*all_secure_linksP)
	deduplicateur(*all_insecure_linksP)*/

	return *all_insecure_linksP, *all_secure_linksP
}

var to_explore_lock sync.RWMutex

func poolcrawl(page string, root string, to_explore *[]string, explored_linksP *[]string, all_insecure_linksP *[]string, all_secure_linksP *[]string, total_links_list_lockP *sync.RWMutex, wg *sync.WaitGroup) {
	defer wg.Done()
	if strings.Contains(page, root) && run {
		insecure_links_page, secure_links_page := crawl(page, root)
		total_links_list_lockP.Lock()
		*all_insecure_linksP = append(*all_insecure_linksP, insecure_links_page...) // on enregistre les liens http
		*all_secure_linksP = append(*all_secure_linksP, secure_links_page...)
		total_links_list_lockP.Unlock()
		/*fmt.Println(len(explored_links))
		fmt.Println(len(secure_links_page))
		fmt.Println(len(*to_explore))*/

		var local_to_add []string
		for _, link := range secure_links_page {
			if strings.Contains(link, root) { // on reste sur le même site
				if !stringInStrings(link, *explored_linksP) { // on va pas 2 fois sur la même page
					local_to_add = append(local_to_add, link)
				}
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
			if strings.Compare(uniq, elem) == 0 { //si on le trouve
				unique = false
			}
		}
		if unique {
			uniques = append(uniques, elem)
		}
	}
	return uniques
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

func sendString(writer *bufio.Writer, texte string) (werror error, flusherror error) {
	_, err := writer.Write([]byte(texte + "\x04"))
	if err != nil {
		print(err)
	}
	err2 := writer.Flush()
	if err2 != nil {
		print(err)
	}
	return err, err2
}
func recvString(reader *bufio.Reader) (string, error) {
	str, errs := reader.ReadString('\x04')
	return strings.TrimSuffix(str, "\x04"), errs
}
