package main

import (
	"fmt"
	"io"
	"net"
)

/*
ports : 1998 -> port d'écoute du reverse proxy pour la destination serveur
1999: port d'écoute pour la source client
*/
func main() {
	for {
		serveur, _ := net.Listen("tcp", ":1998") //serveur
		for {
			l, _ := serveur.Accept()
			fmt.Println("server connected")
			client, _ := net.Listen("tcp", ":1999")
			for {
				c, _ := client.Accept()
				fmt.Println("client connected")
				go io.Copy(l, c)
				go io.Copy(c, l)
			}
		}
	}
}
