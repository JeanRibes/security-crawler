package main

import (
	"./utils"
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var defaulthost = "do-fra1.ribes.ovh:2000"

func main() {

	host := flag.String("host", defaulthost, "Serveur d'indexation à utiliser")
	website := flag.String("website", "https://example.ribes.ovh", "Site web à indexer")
	flag.Parse()

	localServer := findLocalServer()
	println(localServer)
	if localServer != "" {
		if *host == defaulthost {
			host = &localServer //c'est mieux dans ce sens les pointeurs
			//*host = localServer
			fmt.Println([]byte("192.168.22.23:2000"))
			fmt.Println([]byte(*host))
		}
	}

	fmt.Println("Using host " + *host + " and website " + *website)
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	motd, err3 := utils.RecvString(reader) //"bonjour ...."
	if err3 != nil {
		fmt.Println(err3.Error())
		os.Exit(1)
	}
	fmt.Println(motd)

	/*scanner := bufio.NewScanner(os.Stdin)
	if len(os.Args) > 2 {
		lien = os.Args[2]
	} else {
		lien = scanner.Text()
	}
	for true {

		if strings.HasPrefix(lien, "https://") && !strings.HasSuffix(lien, "/") {
			break
		} else {
			fmt.Println(lien + ": lien non valide, il doit commencer par https:// et finir par .tld, sans '/' final")
			scanner.Scan()
			lien = scanner.Text()
		}
	}*/

	err2, _ := utils.SendString(writer, *website)
	//err2, _ := sendString(writer, "https://jean.ribes.ovh")
	if err2 != nil {
		fmt.Println(err2)
		os.Exit(1)
	}
	response, err3 := utils.RecvString(reader)
	if err3 != nil {
		fmt.Println(err3)
		os.Exit(1)
	}
	fmt.Println(response)
	errclose := conn.Close()
	if errclose != nil {
		fmt.Println(errclose)
	}
}

/*
Cherche une addresse en écoutant le traffic broadcast
*/
func findLocalServer() string {
	uconn, err0 := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 2020, Zone: ""})
	err := uconn.SetReadDeadline(time.Now().Add(time.Second * 3))
	if err != nil {
		fmt.Println(err)
	}
	if err0 == nil {
		buf := make([]byte, 30)
		_, _, err1 := uconn.ReadFromUDP(buf)
		if err1 == nil {
			ls := strings.Split(string(buf), "\x04") // il faut un terminateur sinon on lit les 0 du buffer
			return ls[0]
		} else {
			fmt.Println(err1)
		}
	} else {
		fmt.Println(err0)
	}
	return ""
}
