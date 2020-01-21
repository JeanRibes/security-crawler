package main

import (
	"./utils"
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var relative_links_regex = regexp.MustCompile(`href="(?P<link>\S+)"`)

var run = true // permet d'arrêter l'itération des boucles & la récursion

func main() {
	proxy_host := flag.String("proxy-host", "vps.ribes.ovh:1998", "Adresse:Port du reverse proxy TCP")
	bind_addr := flag.String("bind-addr", ":2000", "Adresse:Port sur lequel écouter")
	use_proxy := flag.Bool("no-proxy", false, "Essayer de récupérer des clients depuis un proxy inverse")

	flag.Parse()
	go announceBroadcast(*bind_addr)

	if !*use_proxy {
		go use_reverse_proxy(*proxy_host) // connexion inverse pour récupérer des clients depuis un proxy internet
	}

	ln, err0 := net.Listen("tcp", *bind_addr)
	if err0 != nil {
		fmt.Println(err0)
	}
	fmt.Println("Listening on " + ln.Addr().String())
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}
		go handleClient(conn)
	}
}

func use_reverse_proxy(proxy_host string) {
	should_retry := true
	for should_retry {
		println("boucle reverse proxy ...")
		reverseConn, rerr := net.Dial("tcp", proxy_host) //connexion au reverse proxy qui va forward le traffic tcp vers nous
		if rerr == nil {
			fmt.Println("Connecté au reverse proxy " + reverseConn.RemoteAddr().String() + " avec l'addresse " + reverseConn.LocalAddr().String())
			reader := bufio.NewReader(reverseConn)
			clientip, err := reader.ReadString('\x02') // envoyé à la connexion d'un client
			fmt.Println("client " + clientip + " reçu du reverse proxy")
			if err != nil {
				fmt.Println(err)
			} else {
				go func() {
					handleClient(reverseConn) // bloquant
					errc := reverseConn.Close()
					if errc != nil {
						fmt.Println(errc)
					}
				}()
			}
		} else {
			should_retry = false
		}
		//println("... fini")
	}
}

func handleClient(conn net.Conn) {
	fmt.Println("Service du client " + conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	//writer.WriteString("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04²")
	//writer.WriteString("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04")
	//writer.Write([]byte("Bonjour, bienvenue sur ce serveur. Entrez le site à indexer suivi du caractère ASCII EOT(end-of-transmission)\x04"))
	utils.SendString(writer, "Bonjour, bienvenue sur ce serveur. Donnez le site à indexer sans '/' final")

	//lien, err := reader.ReadString('\x04')
	lien, err := utils.RecvString(reader)
	if err != nil {
		print(err)
		utils.SendString(writer, "Erreur de communication")
	} else {
		pourcentage := to_index(lien)
		if pourcentage > 0 {
			utils.SendString(writer, "Pourcentage de liens non sécurisés : "+strconv.Itoa(pourcentage)+"%")
		} else {
			utils.SendString(writer, "Le site n'a pas pu être indexé")
		}
	}
	//writer.WriteString(strconv.Itoa(to_index(strings.TrimSuffix(lien, "²"))) + "\x04²")
	//writer.Flush()
}
func to_index(website string) int {
	insecure, secure := crawing_loop(website, website)
	return int((100 * float64(len(insecure))) / float64(len(secure)+len(insecure)))
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

/*
Cette fonction orchestre les scrape de pages web, en lançant poolcrawl de manière parallèle.
Elle lance des requêtes sur tous les liens trouvés sur la 1e page, attend que toutes les réponses aient été reçues,
puis fait de même sur tous les liens trouvés dans toutes les pages.
C'est une sorte de récursivité d'un point de vue des liens hypertexte.
Entre les itérations elle enlève les liens présents en double (ça fait vite beaucoup sur les sites avec des menus...)
*/
func crawing_loop(page string, root string) ([]string, []string) {
	wg := &sync.WaitGroup{}
	explored_linksP := &[]string{}
	//explored_links_lock := &sync.RWMutex{}
	total_links_list_lockP := &sync.RWMutex{}
	all_insecure_linksP := &[]string{}
	all_secure_linksP := &[]string{}
	to_explore := &[]string{root}
	profondeur := 0
	for len(*to_explore) > 0 && profondeur < 4 {
		println(profondeur)
		profondeur += len(*to_explore) / 100
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

	return *all_insecure_linksP, *all_secure_linksP
}

var to_explore_lock sync.RWMutex

/*
Cette fonction s'occupe de faire fonctionnel `crawl()` dans un environnement avec des goroutines où on ne peut
pas utliser 'return'
*/
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
		for _, link := range secure_links_page { // on se limite ) 100 liens
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
			continue // en fait si ça commence par https:// ça ne va pas commencer par autre chose
		}
		if strings.Contains(link, "http://") {
			//fmt.Println(" is")
			http_links = append(http_links, link)
			continue // oui on peut coder ça avec des ELSE mais c'est moins lisible
		}
		if link[0] == '?' { //le lien diffère juste des query params url, donc il faut le reconstruire
			// pas sûr que ça permette de trouver plus d'infos à part peut-être sur Wordpress & co
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

/*
Digère une page web et renvoie les liens trouvés dessus (différenciés entre https et http
*/
func crawl(site string, root string) (http_links []string, https_links []string) {
	//fmt.Print("Exploring page " + site + " for links")

	page, err := http.Get(site)
	if err != nil {
		fmt.Println(err)
		return []string{}, []string{}
	}
	if strings.Contains(page.Header.Get("content-type"), "text/html") { // on évite de vomir des erreurs en lisant une image
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
func announceBroadcast(message string) {
	a := strings.Split(message, ":")
	message = a[1]
	addrs, _ := net.InterfaceAddrs()
	var laddr string
	for _, addr := range addrs {
		if addr.String() != "127.0.0.1/8" {
			laddr = addr.String()
			fmt.Println(addr.String())
			break
		}
	}
	h := strings.Split(laddr, "/")
	laddr = h[0]
	buf := []byte(laddr + ":" + message + "\x04")
	broadcastAddr := net.UDPAddr{IP: net.IPv4bcast, Port: 2020, Zone: ""}
	bconn, err0 := net.ListenUDP("udp4", nil)
	if err0 == nil {
		fmt.Println(string(buf))
		_, err1 := bconn.WriteToUDP(buf, &broadcastAddr)
		if err1 == nil {
			for {
				time.Sleep(time.Millisecond * 100) // attend 3s avant de ré-emettre
				bconn.WriteToUDP(buf, &broadcastAddr)
			}
		} else {
			fmt.Println(err1)
		}
	} else {
		fmt.Println(err0)
	}
}
