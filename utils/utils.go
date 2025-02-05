package utils

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"strings"
)

func SendString(writer *bufio.Writer, texte string) (werror error, flusherror error) {
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
func RecvString(reader *bufio.Reader) (string, error) {
	str, errs := reader.ReadString('\x04')
	return strings.TrimSuffix(str, "\x04"), errs
}

type Packet struct {
	TypePaquet int
	Message interface{}
}
const (
	Type_control=0
	Type_craw_request=1
	Type_crawl_response=2
	Type_log=3 //string
)
type Response struct {
	Http_link int
	Https_links int
}

func Send(conn net.Conn, paq Packet) {
	send_buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(send_buf)
	encoder.Encode(paq)
	conn.Write(send_buf.Bytes())
}


func Receive(conn net.Conn) (paq *Packet) {
	rcv_buf := make([]byte, 4096) //il y a peut-être pas besoin d'autant, surtout sur le serveur
	_, err := conn.Read(rcv_buf)
	if err !=nil{
		println(err)
	}
	decoder := gob.NewDecoder(bytes.NewBuffer(rcv_buf))
	errdec := decoder.Decode(paq)
	if errdec != nil {
		fmt.Println(errdec)
	}
	return paq
}