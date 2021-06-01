package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func tailLog(idx int, data chan IPv4) {
	id := Conf.services[idx].service
	file := Conf.services[idx].logName

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
TAILLOOP:
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
				for _, re := range Conf.services[idx].regexps {
					ip := re.FindStringSubmatch(line)
					if len(ip) > 1 {
						addr, err := ParseIPv4(ip[1])
						if err != nil {
							log.Printf("%s: parse address \"%s\" error: %s\n", id, ip[1], err)
							continue
						}
						for _, ignore := range Conf.services[idx].serviceIgnoreIP {
							if ignore.Contains(addr) {
								continue TAILLOOP
							}
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
	last   int64
	expire int64
	count  int
}

func blockTime(count int) int64 {
	return int64(Conf.BanTime << count)
}

func main() {

	err := readConfig("/usr/local/etc/go_f2b.conf")
	if err != nil {
		log.Fatalf("Config parse error: %s\n", err)
	}

	data := make(chan IPv4)
	for i := range Conf.services {
		go tailLog(i, data)
	}

	db := make(map[IPv4]Entry)
	ticker := time.NewTicker(10 * time.Second)

MAINLOOP:
	for {
		select {
		case ip := <-data:
			for _, ignore := range Conf.globalIgnoreIP {
				if ignore.Contains(ip) {
					continue MAINLOOP
				}
			}

			now := time.Now().Unix()
			entry := db[ip]

			if entry.expire > now {
				log.Println("dup count or block failed?", ip)
				continue
			}

			entry.count++
			entry.last = now
			//if entry.count < 10 {
			//	entry.count++
			//}
			if entry.count >= Conf.MaxRetry {
				bt := blockTime(entry.count - Conf.MaxRetry)
				entry.expire = now + bt
				log.Println("blocking", ip, "for", bt)
				cmd := exec.Command("/sbin/ipfw", "table", "blacklist", "add", ip.String())
				err = cmd.Run()
				if err != nil {
					log.Println("exec error:", err)
				}
			}
			db[ip] = entry

		case <-ticker.C:
			now := time.Now().Unix()
			for ip, entry := range db {
				if entry.expire != 0 && entry.expire < now {
					entry.expire = 0
					log.Println("unblocking", ip)
					cmd := exec.Command("/sbin/ipfw", "table", "blacklist", "delete", ip.String())
					err = cmd.Run()
					if err != nil {
						log.Println("exec error:", err)
					}
				}
				db[ip] = entry
				if entry.last+Conf.DBPurgeAge < now && entry.expire == 0 {
					log.Println("Purging entry:", ip.String())
					delete(db, ip)
				}
			}
		}
	}
}
