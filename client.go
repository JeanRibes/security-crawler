package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	fmt.Print("\x02a\x03")
	conn, err := net.Dial("tcp", "localhost:1984")
	if err != nil {
		print(err)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	motd, err := reader.ReadString('²')
	//motd, err := reader.ReadString('\x04')
	fmt.Println(motd)
	//_, err2 := fmt.Fprintf(conn, "https://jean.ribes.ovh\x04")
	_, err2 := writer.WriteString("https://jean.ribes.ovh²\x04")
	writer.Flush()
	if err2 != nil {
		print(err2)
	}
	response, err := reader.ReadString('²')
	//response, err := reader.ReadString('\x04')
	fmt.Println(response)
}
