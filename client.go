package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:1984")
	if err != nil {
		print(err)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(os.Stdin)
	motd, err3 := recvString(reader) //"bonjour ...."
	if err3 != nil {
		print(err3)
	}
	fmt.Println(motd)
	var lien string
	for scanner.Scan() {
		lien = scanner.Text()
		if strings.HasPrefix(lien, "https://") && !strings.HasSuffix(lien, "/") {
			break
		} else {
			fmt.Println(lien + ": lien non valide, il doit commencer par https:// et finir par .tld, sans '/' final")
		}
	}

	err2, _ := sendString(writer, lien)
	//err2, _ := sendString(writer, "https://jean.ribes.ovh")
	if err2 != nil {
		print(err2)
	}
	response, err := recvString(reader)
	fmt.Println(response)
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
