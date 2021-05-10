package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"
)

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
						addr, err := ParseIPv4(ip[1])
						if err != nil {
							log.Printf("%s: parse address \"%s\" error: %s\n", id, ip[1], err)
							continue
						}
						data <- addr
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

const MAXHISTORYTIME = 3600 * 84 * 7 // 1 week

func blocktime(count int) int64 {
	return int64(100 << count)
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
	whiteNet, _ := ParseIPv4("192.168.1.0")
	whiteMask := IPv4(0xFFFFFF00)
	for {
		s := <-data
		if s&whiteMask == whiteNet {
			continue
		}
		now := time.Now().Unix()
		entry, ok := db[s]

		if !ok {
			entry = Entry{first: now}
		}

		if entry.expire > now { // dup count or block failed
			log.Println("dup count or block failed?", s)
			continue
		}

		if entry.count < 10 {
			entry.count++
		}
		bt := blocktime(entry.count)
		entry.last = now
		entry.expire = now + bt
		log.Println("blocking", s, "for", bt)
		cmd := exec.Command("/sbin/ipfw", "table", "blacklist", "add", s.String())
		err := cmd.Run()
		if err != nil {
			log.Println("exec error:", err)
		}

		db[s] = entry
	}
}
