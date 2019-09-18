package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
)

var httpMethods = []string{
	"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT",
}

// global packet sequence id
var id int64

type httpParser struct{}

const XML_FORMAT string = "xml"
const JSON_FORMAT string = "json"
const HEADER_TRANSER_ENCODING string = "Transfer-Encoding"
const HEADER_CONTENT_TYPE string = "Content-Type"
const HEADER_UPGRADE string = "Upgrade"
const HEADER_VAL_CHUNKED string = "chunked"
const HEADER_VAL_WEBSOCKET string = "websocket"

const HTTP_DIRECTION_REQUEST string = "Request"
const HTTP_DIRECTION_RESPONSE string = "Response"
const LINEBREAK string = "\r\n"

// the concerned content of a http request/response
type HttpPacket struct {
	PacketId string

	// net/transport layer information
	HostSrc string
	HostDst string
	PortSrc string
	PortDst string

	// first line of the http payload
	HeadLine string

	Headers map[string]string
	Body    string

	// if the Transfer-Encoding header is chunked
	IsTransferEncodingTrunck bool

	// json or xml depending on the Content-Type header
	bodyFormat string

	// request or response
	Direction string

	// if it is a websocket request/response
	IsWebSocket bool
}

func (packet *HttpPacket) init() {
	packet.Headers = make(map[string]string, 10)
	packet.IsTransferEncodingTrunck = false
	packet.IsWebSocket = false

	int64Id := atomic.AddInt64(&id, 1)
	packet.PacketId = strconv.FormatInt(int64Id, 10)
}

func (packet *HttpPacket) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(packet.HeadString())

	if packet.IsBodyJson() {
		var str bytes.Buffer
		err := json.Indent(&str, []byte(packet.Body), "", "    ")
		if err != nil || !prettyJson {
			buffer.WriteString(packet.Body)
		} else {
			buffer.WriteString(str.String())
		}
	} else if packet.IsBodyXml() {
		xmlPrettyString, err := xmlPrettify(packet.Body)
		if err != nil {
			buffer.WriteString(packet.Body)
			log.Println(err)
		} else {
			buffer.WriteString(xmlPrettyString)
		}
	} else {
		buffer.WriteString(packet.Body)
	}
	buffer.WriteString(LINEBREAK)

	return buffer.String()
}

func (packet *HttpPacket) HeadString() string {
	var buffer bytes.Buffer
	buffer.WriteString("---------------------- [")
	buffer.WriteString(packet.PacketId)
	buffer.WriteString("] ")
	buffer.WriteString(packet.Direction)
	buffer.WriteString(" ----------------------")
	buffer.WriteString(LINEBREAK)

	buffer.WriteString(packet.HostSrc + ":" + packet.PortSrc + " -> " + packet.HostDst + ":" + packet.PortDst)
	buffer.WriteString(LINEBREAK)
	buffer.WriteString(LINEBREAK)

	buffer.WriteString(packet.HeadLine)
	buffer.WriteString(LINEBREAK)

	for e := range packet.Headers {
		buffer.WriteString(e + ": " + packet.Headers[e])
		buffer.WriteString(LINEBREAK)
	}

	buffer.WriteString(LINEBREAK)

	return buffer.String()
}

func (packet *HttpPacket) addHeader(k string, v string) {

	packet.Headers[k] = v
	if strings.EqualFold(k, HEADER_TRANSER_ENCODING) && strings.EqualFold(v, HEADER_VAL_CHUNKED) {

		packet.IsTransferEncodingTrunck = true

	} else if strings.EqualFold(k, HEADER_CONTENT_TYPE) {

		if strings.Contains(strings.ToLower(v), JSON_FORMAT) {
			packet.bodyFormat = JSON_FORMAT
		} else if strings.Contains(strings.ToUpper(v), XML_FORMAT) {
			packet.bodyFormat = XML_FORMAT
		}

	} else if strings.EqualFold(k, HEADER_UPGRADE) && strings.EqualFold(v, HEADER_VAL_WEBSOCKET) {

		packet.IsWebSocket = true

	}
}

func (packet *HttpPacket) IsBodyXml() bool {
	return strings.EqualFold(packet.bodyFormat, XML_FORMAT)
}
func (packet *HttpPacket) IsBodyJson() bool {
	return strings.EqualFold(packet.bodyFormat, JSON_FORMAT)
}

func (h *httpParser) New(net, transport gopacket.Flow) tcpassembly.Stream {
	r := tcpreader.NewReaderStream()

	go process(r, net, transport)

	return &r
}

func process(stream tcpreader.ReaderStream, net, transport gopacket.Flow) {

	buf := bufio.NewReader(&stream)

	packet := new(HttpPacket)
	packet.init()

	packet.HostSrc = net.Src().String()
	packet.HostDst = net.Dst().String()
	packet.PortSrc = transport.Src().String()
	packet.PortDst = transport.Dst().String()

	idx := 0
	isInBody := false
	bodyIdx := 0

	for {
		l, _, err := buf.ReadLine()
		if err != nil {
			break
		}
		line := string(l)

		if idx == 0 {

			upperLine := strings.ToUpper(line)
			if strings.Index(upperLine, "HTTP") == 0 {
				// response
				packet.Direction = HTTP_DIRECTION_RESPONSE
			} else if startsWithValidHttpMethod(upperLine) {
				// request
				packet.Direction = HTTP_DIRECTION_REQUEST
			} else {
				// not http
				break
			}
			packet.HeadLine = line

		} else if isInBody {

			// start read body
			if !packet.IsTransferEncodingTrunck || (bodyIdx%2 == 1) {

				packet.Body = packet.Body + LINEBREAK + (line)

				// websocket frame will be printed once it received
				if packet.IsWebSocket {
					log.Println("[" + packet.PacketId + "] " + line)
				}
			}

			bodyIdx++
		} else {
			if (!isInBody) && len(line) == 0 {

				// body starts
				isInBody = true
				if packet.IsWebSocket {
					log.Println(packet.HeadString())
				}

			} else {
				// headers
				if strings.Contains(line, ":") {

					hs := strings.Split(line, ":")
					k := strings.TrimSpace(hs[0])
					v := strings.TrimSpace(hs[1])

					packet.addHeader(k, v)
				}
			}
		}

		idx++
	}

	if !packet.IsWebSocket {
		log.Println(packet.String())
	}
}

func startsWithValidHttpMethod(l string) bool {
	for e := range httpMethods {
		if strings.Index(l, httpMethods[e]) >= 0 {
			return true
		}
	}
	return false
}
