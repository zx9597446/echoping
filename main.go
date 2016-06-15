package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var remoteAddr = flag.String("r", "", "remote addr")
var payloadSize = flag.Int("s", 64, "payload size")
var interval = flag.Int64("i", 100, "interval in milliseconds")
var count = flag.Int("c", 50, "send count")
var number = flag.Int("n", 1, "how many connections")
var protocol = flag.String("p", "tcp", "protocol: tcp or udp")
var genCharts = flag.Bool("g", false, "generate charts")

var payload string

const headSize = 15

type Result struct {
	max, min, avg int
	data          []int
}

func init() {
	payload = randString(*payloadSize)
}

func encodePacket() []byte {
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

func handleReceive(conn net.Conn, notify chan Result) {
	bs := make([]byte, headSize)
	ps := make([]byte, *payloadSize)
	all := int64(0)
	max := 0
	min := 0xFFFFFFFFFFFFFFF
	avg := (0)
	data := make([]int, *count)
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
		data[i] = elapsed
		_, err = io.ReadFull(conn, ps)
		if err != nil {
			log.Println(err)
			break
		}
		all = all + int64(elapsed)
		avg = int(all / int64(i+1))
	}
	notify <- Result{max, min, avg, data}
}

func runOne(remoteAddr string, notify chan Result) {
	conn, err := net.Dial(*protocol, remoteAddr)
	if err != nil {
		log.Panicln(err)
	}
	defer conn.Close()
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
}

func makeCharts(idx int, data []int) {
	t := time.Now()
	now := fmt.Sprintf("%d-%d-%dH%dM%dS%d-%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), idx)
	const header = `
# The chart type , option : spline/line/bar/column/area
ChartType = spline
Title = packet RTT latency (ms)
SubTitle = %s
ValueSuffix = ms 

# The x Axis numbers. The count this numbers MUST be the same with the data series
XAxisNumbers = %s

# The y Axis text
YAxisText = Latency (ms)

# The data and the name of the lines
Data|Latency = %s
`
	x := []string{}
	d := []string{}
	for i := 0; i < len(data); i++ {
		x = append(x, strconv.Itoa(i))
		d = append(d, strconv.Itoa(data[i]))
	}
	sx := strings.Join(x, ", ")
	sd := strings.Join(d, ", ")

	all := fmt.Sprintf(header, now, sx, sd)
	ioutil.WriteFile(now+".chart", []byte(all), os.ModePerm)
}

func main() {
	flag.Parse()
	results := make(chan Result, *number)
	wg := &sync.WaitGroup{}
	for i := 0; i < *number; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runOne(*remoteAddr, results)
		}()
	}
	wg.Wait()
	for i := 0; i < *number; i++ {
		ret := <-results
		if *genCharts {
			makeCharts(i, ret.data)
		}
		log.Printf("RTT min: [%d] ms, RTT max: [%d] ms, RTT avg: [%d] ms\n", ret.min, ret.max, ret.avg)
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
