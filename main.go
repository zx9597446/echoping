package main

import (
	"flag"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

var remoteAddr = flag.String("r", "", "remote addr")
var payloadSize = flag.Int("s", 32, "payload size")
var interval = flag.Int64("i", 1, "interval in seconds")
var count = flag.Int("c", 1, "send count")

func encodePacket() []byte {
	payload := randString(*payloadSize)
	sendTime, _ := time.Now().MarshalBinary()
	sendTime = append(sendTime, payload...)
	return sendTime
}

func decodeLatency(bs []byte) time.Duration {
	before := time.Time{}
	err := before.UnmarshalBinary(bs)
	if err != nil {
		log.Fatal(err)
	}
	return time.Since(before)
}

func handleReceive(conn net.Conn) {
	bs := make([]byte, 15)
	ps := make([]byte, *payloadSize)
	i := 0
	min := 0xFFFFFFFFFFF
	max := 0
	all := int64(0)
	for {
		_, err := io.ReadFull(conn, bs)
		if err != nil {
			log.Println(err)
			break
		}
		elapsed := decodeLatency(bs) * time.Millisecond
		iElapsed := int(elapsed)
		if iElapsed > max {
			max = iElapsed
		}
		if iElapsed < min {
			min = iElapsed
		}
		all = all + int64(iElapsed)
		log.Printf("packet RTT: [%d] ms", elapsed)
		_, err = io.ReadFull(conn, ps)
		if err != nil {
			log.Println(err)
			break
		}
		i = i + 1
	}
	log.Println("RTT min: %d, RTT max: %d, RTT avg: %d\n", min, max, all/int64(i))
}

func main() {
	flag.Parse()
	conn, err := net.Dial("tcp", *remoteAddr)
	if err != nil {
		log.Panicln(err)
	}
	defer conn.Close()
	go handleReceive(conn)
	for i := 0; i < *count; i++ {
		buf := strings.NewReader(string(encodePacket()))
		_, err := io.CopyN(conn, buf, int64(buf.Len()))
		if err != nil {
			log.Println(err)
			break
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
