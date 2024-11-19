package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		if conn != nil {
			go handleRequest(conn)

		}
	}

}

func handleRequest(c net.Conn) {
	var reqBody string
	reqBytes := make([]byte, 1024)
	_, err := c.Read(reqBytes)
	request := strings.TrimSpace(string(reqBytes))
	r := strings.Split(request, "\r\n")
	requestLine := strings.Split(r[0], " ")
	path := requestLine[1]
	method := strings.TrimSpace(requestLine[0])
	reqBodyBuff := strings.Split(request, "\r\n\r\n")

	headers := map[string]string{}
	for i := 1; i < len(r); i++ {
		header := r[i]
		h := strings.Split(header, ": ")
		if len(h) == 2 {
			headers[h[0]] = h[1]
		}
	}
	contentLength, ok := headers["Content-Length"]
	if ok {
		_, err := strconv.Atoi(contentLength)
		if err != nil {
			fmt.Println("Error parsing content length: ", err)
			c.Close()
		}
		reqBody = strings.TrimSpace(reqBodyBuff[1])
		fmt.Println(reqBody)
	}
	if err != nil {
		fmt.Print("Error reading connection: ", err)
	}
	u, uri := formatUri(path)
	fmt.Println(method)

	switch method {
	case "POST":
		filename := uri[0]
		switch u {
		case "files":
			file, err := os.Create(fmt.Sprintf("./tmp/%v.txt", filename))
			defer file.Close()
			fmt.Println(reqBody)
			if err != nil {
				fmt.Println("Error creating file: ", err)
			}
			content := fmt.Sprintf("%s\n", reqBody)
			fmt.Println(content)
			_, err = file.Write([]byte(content))
			if err != nil {
				fmt.Println("Error writing file: ", err)
				c.Write(notFound())
				c.Close()
			}
			file.Sync()
			file.Close()
			handleResponse(c, []byte("HTTP/1.1 201 Create\r\n\r\n"))
		}

	case "GET":
		if path == "/" {
			handleResponse(c, formatReponse(nil, ""))
		}

		switch u {
		case "echo":
			if len(uri) > 0 {
				response := uri[0]
				content := strings.TrimSpace(response)
				header := formatHeaders("text/plain", len(content))
				encodings, ok := headers["Accept-Encoding"]
				if ok {
					encode := strings.Split(encodings, ",")
					var scheme string
					for i := range len(encode) {
						encode[i] = strings.TrimSpace(encode[i])
						if encode[i] == "gzip" {
							scheme = "gzip"
						}
					}
					switch scheme {
					case "gzip":
						var buff bytes.Buffer
						writer := gzip.NewWriter(&buff)
						_, err := writer.Write([]byte(content))
						writer.Close()
						content = buff.String()
						if err != nil {
							fmt.Println("Error writting to buffer: ", err)
						}
						handleResponse(c, []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n%v", len(content), content)))

					default:
						handleResponse(c, formatReponse(header, content))
					}
				} else {
					handleResponse(c, formatReponse(header, content))
				}
			} else {
				handleResponse(c, notFound())
			}
		case "user-agent":
			content := headers["User-Agent"]
			header := formatHeaders("text/plain", len(content))
			handleResponse(c, formatReponse(header, content))

		case "files":
			filename := uri[0]
			switch path {
			case "filename":

				file, err := os.Open(fmt.Sprintf("./tmp/%v.txt", filename))
				if err != nil {
					fmt.Println("Error opening file: ", err)
					handleResponse(c, notFound())
				}
				var buffRead []byte
				n, err := file.Read(buffRead)
				fmt.Println(n)
				if err != nil {
					fmt.Println("Error reading file: ", err)
					handleResponse(c, notFound())
				}
				content := string(buffRead)
				header := formatHeaders("application/octet-stream", n)
				handleResponse(c, formatReponse(header, content))

			}

		default:
			fmt.Println()
			handleResponse(c, notFound())
		}
	}

}
func handleResponse(conn net.Conn, msg []byte) {
	conn.Write(msg)
	conn.Close()
}
func formatHeaders(contentType string, length int) map[string]string {
	return map[string]string{
		"Content-Type":   contentType,
		"Content-Length": fmt.Sprintf("%d", length),
	}
}
func formatReponse(headers map[string]string, content string) []byte {
	if headers == nil {
		return []byte("HTTP/1.1 200 OK\r\n\r\n")
	}
	var header string
	for k, v := range headers {
		header += fmt.Sprintf("%v: %s\r\n", k, v)
	}
	return []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\n%v\r\n%s", header, content))
}

func formatUri(uri string) (string, []string) {
	var root string
	var path []string

	uri = strings.TrimSpace(uri)
	u := strings.Split(uri, "/")
	if len(u) == 1 {
		root = u[1]
	}
	if len(u) > 1 {
		root := u[1]
		path := u[2:]
		return root, path
	}
	return root, path
}

func notFound() []byte {
	return []byte("HTTP/1.1 404 NOT FOUND\r\n\r\n")
}
