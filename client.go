package main

import (
	"./utils"
	"bufio"
	"flag"
	"fmt"
	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
	"net"
	"os"
	"strings"
	"time"
)

var defaulthost = "do-fra1.ribes.ovh:2000"
var defaultwebsite = "https://example.ribes.ovh"

var mainwin *ui.Window

func main() {
	ui.Main(setupUI)
}

func setupUI() {
	mainwin = ui.NewWindow("Security Crawler", 640, 130, true)
	mainwin.OnClosing(func(*ui.Window) bool {
		ui.Quit()
		return true
	})
	ui.OnShouldQuit(func() bool {
		mainwin.Destroy()
		return true
	})
	mainwin.SetMargined(true)

	vbox := ui.NewVerticalBox()
	website_hbox := ui.NewHorizontalBox()
	server_hbox := ui.NewHorizontalBox()
	mainwin.SetChild(vbox)

	server_entry := ui.NewEntry()
	server_entry.SetText(defaulthost)
	website_entry := ui.NewEntry()
	website_entry.SetText(defaultwebsite)
	button := ui.NewButton("Lancer l'exploration du site !")

	button.OnClicked(func(*ui.Button) {
		ui.MsgBox(mainwin, "Réponse du serveur", crawl_website(server_entry.Text(), website_entry.Text()))
	})

	server_hbox.Append(ui.NewLabel("Adresse du serveur :"), false)
	server_hbox.Append(server_entry, true)
	website_hbox.Append(ui.NewLabel("Adresse du site :"), false)
	website_hbox.Append(website_entry, true)

	vbox.Append(server_hbox, false)
	vbox.Append(website_hbox, false)
	vbox.Append(button, false)

	mainwin.Show()
}

func crawl_website(server_adress string, website_adress string) string {
	host := flag.String("host", server_adress, "Serveur d'indexation à utiliser")
	website := flag.String("website", website_adress, "Site web à indexer")
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

	errclose := conn.Close()
	if errclose != nil {
		fmt.Println(errclose)
	}
	fmt.Println(response)
	return response
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
