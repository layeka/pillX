package pillx

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

type WebSocketHeader struct {
	OpcodeByte  byte
	ProloadByte byte
}

type WebSocketProtocol struct {
	Header  *WebSocketHeader
	Content []byte
}

func (websocket *WebSocketProtocol) Analyze(client *Response) (err error) {
	var err error
	header := new(WebSocketHeader)
	websocket.Header = header
	buf := client.conn.buf

	//读取fin
	header.OpcodeByte, err = buf.ReadByte()
	fin := header.fin[0] >> 7
	if fin == 0 {

	}

	//读取opcode
	opcode, err = header.OpcodeByte & 0x0f
	if opcode == 8 {
		log.Print("Connection closed")
		//self.Close()
		break
	}

	header.ProloadByte, err = buf.ReadByte()
	mask := header.ProloadByte >> 7
	proload := header.ProloadByte & 0x7f

	var (
		lengthBytes  []byte
		length       uint64
		l            uint16
		maskKeyBytes []byte
		contentBuf   []byte
	)

	//读取长度
	switch {
	case payload < 126:
		length = uint64(payload)

	case payload == 126:
		lengthBytes = make([]byte, 2)
		buf.Read(lengthBytes)
		binary.Read(bytes.NewReader(lengthBytes), binary.BigEndian, &l)
		length = uint64(l)

	case payload == 127:
		lengthBytes = make([]byte, 8)
		buf.Read(lengthBytes)
		binary.Read(bytes.NewReader(lengthBytes), binary.BigEndian, &length)
	}

	if mask == 1 {
		maskKeyBytes = make([]byte, 4)
		buf.Read(maskKeyBytes)
	}

	contentBuf = make([]byte, length)
	buf.Read(contentBuf)

	if mask == 1 {
		//解码内容
		for i, v := range contentBuf {
			contentBuf[i] = v ^ mkey[i%4]
		}
	}
	websocket.Content = contentBuf
	client.callbackServe(SYS_ON_MESSAGE)

	return nil
}

func (gateway *WebSocketProtocol) Encode(msg interface{}) (buf []byte, err error) {
	buff := new(bytes.Buffer)
	//binary.Write(buff, binary.BigEndian, msg.(*GateWayProtocol).Header)

	frame := []byte{129}

	data := msg.(*GateWayProtocol).Content
	length := len(data)

	switch {
	case length < 126:
		frame = append(frame, byte(length))
	case length <= 0xffff:
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		frame = append(frame, byte(126))
		frame = append(frame, buf...)
	case uint64(length) <= 0xffffffffffffffff:
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		frame = append(frame, byte(127))
		frame = append(frame, buf...)
	default:
		log.Fatal("Data too large")
		return false
	}
	frame = append(frame, data...)

	binary.Write(buff, binary.BigEndian, frame)
	//MyLog().Info(buff.Bytes())
	return buff.Bytes(), nil
}

func (self *WebSocketProtocol) Handshake() []byte {
	if self.Shook {
		return true
	}
	reader := bufio.NewReader(self.Conn)
	key := ""
	str := ""
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Fatal(err)
			return false
		}
		if len(line) == 0 {
			break
		}
		str = string(line)
		if strings.HasPrefix(str, "Sec-WebSocket-Key") {
			key = str[19:43]
		}
	}
	sha := sha1.New()
	io.WriteString(sha, key+"258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	key = base64.StdEncoding.EncodeToString(sha.Sum(nil))

	header := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Accept: " + key + "\r\n" +
		"Upgrade: websocket\r\n\r\n"

	return []byte(header)
}
