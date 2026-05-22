package tool

import (
	"net"
	"strings"
	"unicode"
)

func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func IsUnicodeLetterOrDigit(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return len(s) > 0
}

func HasSQLRisk(s string) bool {
	for _, r := range s {
		switch r {
		case '\'', '"', ';', '\\', '#':
			return true
		}
	}
	if strings.Contains(s, "--") || strings.Contains(s, "/*") {
		return true
	}
	return false
}
