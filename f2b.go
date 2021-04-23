package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type IPv4 uint32

func parseIPv4(ip string) IPv4 {
	var ipv4 IPv4 = 0
	for i := 0; i < 4; i++ {
		if len(ip) == 0 {
			return ipv4
		}
		if i > 0 {
			if ip[0] != '.' {
				return ipv4
			}
			ip = ip[1:]
		}
		n, c, ok := dtoi(ip)
		if !ok {
			return ipv4
		}
		ip = ip[c:]
		ipv4 = ipv4<<8 + IPv4(n&0xFF)
	}
	//if len(s) != 0 {
	//	return nil
	//}
	return ipv4
}

func (ip IPv4) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip>>16&0xFF, ip>>8&0xFF, ip&0xFF)
}

func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n > 255 {
			return 255, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}

func tailLog(file string, regexps []string, data chan IPv4) {
	id := path.Base(file)

	var reCompiled []*regexp.Regexp
	for _, re := range regexps {
		newRe, err := regexp.Compile(re)
		if err == nil {
			reCompiled = append(reCompiled, newRe)
		}
	}
	if len(reCompiled) == 0 {
		log.Fatal("no regexp to match!")
	}

	fstat := syscall.Stat_t{}
	if err := syscall.Stat(file, &fstat); err != nil {
		log.Fatal(err)
	}
	inode := fstat.Ino
	log.Printf("%s: inode: %d\n", id, inode)

	fn, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer fn.Close()

	c := 0
	curOffset, err := fn.Seek(0, 2)
	var fi os.FileInfo
	for {
		time.Sleep(200 * time.Millisecond)

		fi, err = fn.Stat()
		if err != nil {
			log.Fatal(err)
		}
		if fi.Size() > curOffset {
			var b strings.Builder
			n, _ := io.Copy(&b, fn)
			//log.Println(n)
			curOffset += n
			for _, line := range strings.Split(b.String(), "\n") {
				for _, re := range reCompiled {
					ip := re.FindStringSubmatch(line)
					if len(ip) > 1 {
						//data <- fmt.Sprintf("%s: %s", id, ip[1])
						data <- parseIPv4(ip[1])
					}
				}
			}
		}

		c++
		if c < 5 {
			continue
		}
		c = 0
		if err = syscall.Stat(file, &fstat); err != nil {
			log.Fatal(err)
		}
		if inode != fstat.Ino { // file was rotated?
			inode = fstat.Ino
			log.Printf("%s: inode change: %d\n", id, inode)
			fn.Close()
			fn, err = os.OpenFile(file, os.O_RDONLY, 0644)

			if err != nil {
				log.Fatal(err)
			}
			curOffset = 0
		}
	}
}

type Entry struct {
	//ip IPv4
	first  int64 // unixtime
	last   int64
	expire int64
	count  int
}

func main() {
	data := make(chan IPv4)
	db := make(map[IPv4]Entry)
	go tailLog("/jails/mail/var/log/maillog",
		[]string{"AUTH failure .* relay=.*\\[(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})\\]$"},
		data)
	go tailLog("/var/log/auth.log",
		[]string{"Invalid (?:user|admin) (?:.*) from (\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}) port",
			"Received disconnect from (\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}) port \\d{1,5}:11: Bye Bye \\[preauth\\]$",
			"Connection closed by (?:authenticating|invalid) user \\S+ (\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}) port \\d{1,5} \\[preauth\\]$"},
		data)
	go tailLog("/jails/mail/var/log/messages",
		[]string{"(?:pop3s|imaps) .* failed:.*\\[(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})\\]$"},
		data)
	//var s string
	for {
		s := <-data
		entry, ok := db[s]
		now := time.Now().Unix()
		if !ok {
			entry = Entry{
				first:  now,
				last:   now,
			}
		}
		if now - entry.last < 3 && ok {
			continue
		}
		entry.count++


		log.Println(s)
	}
}
