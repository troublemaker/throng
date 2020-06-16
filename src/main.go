package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

//time
var doneBy time.Time

//sync
var wg sync.WaitGroup
var runEnd uint32

//stats
var successful_reqs uint64
var connect_fails uint64
var io_fails uint64

//params
var connAddr string
var connTimeout int
var runConnections int
var runDuration int
var messageLen int

//TO DO:  variable length
var message = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}
var recvbuff = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func init() {
	flag.IntVar(&runConnections, "c", 10, "number of connections")
	flag.IntVar(&runDuration, "d", 5, "duration (seconds)")
	flag.IntVar(&connTimeout, "t", 5, "connection operations timeout")

	//TO DO:
	//flag.IntVar(&messageLen, "l", 16, "message length")
	messageLen = 16

	flag.StringVar(&connAddr, "h", "127.0.0.1:7777", " target server address")
	flag.Parse()

	//deadline for all connections
	doneBy = time.Now().Add(time.Second * time.Duration(runDuration+connTimeout))

}

func connection_worker(n int) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", connAddr, time.Second*time.Duration(connTimeout))

	if err != nil {
		log.Println(err)
		atomic.AddUint64(&connect_fails, 1)
		return
	}

	err = conn.SetDeadline(doneBy)
	if err != nil {
		log.Println(err)
		atomic.AddUint64(&connect_fails, 1)
		return
	}

	// start IO loop
	for {
		for done := 0; done < messageLen; {
			r, err := conn.Write(message)
			done += r
			if err != nil {
				log.Println(err)
				atomic.AddUint64(&io_fails, 1)
				return
			}
		}

		for done := 0; done < messageLen; {
			r, err := conn.Read(recvbuff)
			done += r
			if err != nil {
				log.Println(err)
				atomic.AddUint64(&io_fails, 1)
				return
			}
		}

		atomic.AddUint64(&successful_reqs, 1)

		if atomic.LoadUint32(&runEnd) == 1 {
			break
		}
	}
	//
}

func main() {
	flag.Parse()
	fmt.Println("TCP echo load testing.  ")

	//progress animation
	wg.Add(1)
	go func() {
		defer wg.Done()
		progress := "|/-\\"
		i := 0
		for {
			fmt.Printf("\033[%dD", 2)
			fmt.Printf("%c", progress[i])
			fmt.Printf("\033[%dC", 1)
			i++
			if i == 4 {
				i = 0
			}
			time.Sleep(time.Millisecond * 100)
			if atomic.LoadUint32(&runEnd) == 1 {
				break
			}
		}
	}()

	//lauch echo workers
	for i := 0; i < runConnections; i++ {
		wg.Add(1)
		go connection_worker(i)
	}

	time.Sleep(time.Second * time.Duration(runDuration))
	atomic.StoreUint32(&runEnd, 1)
	wg.Wait()

	fmt.Println("Done. stats: \n------------------------------- ")
	fmt.Printf("  Successful operations: %d \n  Failed requests: %d \n  Failed IO: %d \n", successful_reqs, connect_fails, io_fails)
	fmt.Println("------------------------------- ")
	fmt.Printf("  Operations/sec: %d \n", successful_reqs/uint64(runDuration))
	fmt.Println("=============================== ")

}
