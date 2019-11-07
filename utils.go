package main

import (
	"bufio"
	"strings"
)

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
