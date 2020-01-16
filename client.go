package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	host := flag.String("host", "vps.ribes.ovh:1999", "Serveur d'indexation à utiliser")
	website := flag.String("website", "https://example.ribes.ovh", "Site web à indexer")
	flag.Parse()
	fmt.Println("Using host " + *host + " and website " + *website)
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	motd, err3 := recvString(reader) //"bonjour ...."
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

	err2, _ := sendString(writer, *website)
	//err2, _ := sendString(writer, "https://jean.ribes.ovh")
	if err2 != nil {
		fmt.Println(err2)
		os.Exit(1)
	}
	response, err3 := recvString(reader)
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
