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
var payloadSize = flag.Int("s", 64, "payload size")
var interval = flag.Int64("i", 50, "interval in milliseconds")
var count = flag.Int("c", 50, "send count")

var min = 0xFFFFFFFFFFF
var max = 0
var avg = int64(0)

func encodePacket() []byte {
	payload := randString(*payloadSize)
	sendTime, _ := time.Now().MarshalBinary()
	sendTime = append(sendTime, payload...)
	return sendTime
}

func decodeLatency(bs []byte) int {
	before := time.Time{}
	err := before.UnmarshalBinary(bs)
	if err != nil {
		log.Fatal(err)
	}
	t := time.Since(before) / time.Millisecond
	return int(t)
}

func handleReceive(conn net.Conn, notify chan struct{}) {
	bs := make([]byte, 15)
	ps := make([]byte, *payloadSize)
	all := int64(0)
	for i := 0; i < *count; i++ {
		_, err := io.ReadFull(conn, bs)
		if err != nil {
			log.Println(err)
			break
		}
		elapsed := decodeLatency(bs)
		if elapsed > max {
			max = elapsed
		}
		if elapsed < min {
			min = elapsed
		}
		log.Printf("[%d] packet RTT: [%d] ms", i, elapsed)
		_, err = io.ReadFull(conn, ps)
		if err != nil {
			log.Println(err)
			break
		}
		all = all + int64(elapsed)
		avg = all / int64(i+1)
	}
	notify <- struct{}{}
}

func main() {
	flag.Parse()
	conn, err := net.Dial("tcp", *remoteAddr)
	if err != nil {
		log.Panicln(err)
	}
	defer conn.Close()
	notify := make(chan struct{}, 1)
	go handleReceive(conn, notify)
	for i := 0; i < *count; i++ {
		buf := strings.NewReader(string(encodePacket()))
		_, err := io.CopyN(conn, buf, int64(buf.Len()))
		if err != nil {
			log.Println(err)
			break
		}
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
	<-notify
	log.Printf("RTT min: [%d] ms, RTT max: [%d] ms, RTT avg: [%d] ms\n", min, max, avg)
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
