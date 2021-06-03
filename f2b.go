package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
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
			_ = fn.Close()
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
		log.Fatalf("Config load error: %s\n", err)
	}

	var dumpNames [DUMPCOUNT]string
	dumpNames[0] = path.Join(Conf.DBDumpPath, DUMPNAME)
	for i := 1; i < DUMPCOUNT; i++ {
		dumpNames[i] = fmt.Sprintf("%s.%d", dumpNames[0], i)
	}

	db := make(map[IPv4]Entry)
	readDB(db, dumpNames)

	data := make(chan IPv4)
	for i := range Conf.services {
		go tailLog(i, data)
	}

	ticker := time.NewTicker(10 * time.Second)
	statTicker := time.NewTicker(15 * time.Minute)

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

		case <-statTicker.C:
			total, blacks, maxCounts := len(db), 0, 0
			for _, entry := range db {
				if maxCounts < entry.count {
					maxCounts = entry.count
				}
				if entry.expire > 0 {
					blacks++
				}
			}
			log.Printf("stat: %d total records, %d blocked, %d max level\n", total, blacks, maxCounts)

			for i := DUMPCOUNT - 1; i > 0; i-- {
				_ = os.Rename(dumpNames[i-1], dumpNames[i])
			}
			fh, err := os.OpenFile(dumpNames[0], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				continue
			}
			for ip, e := range db {
				_, err = fmt.Fprintf(fh, "%d %d %d %d\n", uint32(ip), e.last, e.expire, e.count)
				if err != nil {
					log.Println(err)
				}
			}
			_ = fh.Close()
		}
	}
}

func readDB(db map[IPv4]Entry, dumpNames [DUMPCOUNT]string) {
	var ip IPv4
	var e Entry
	for _, filename := range dumpNames {
		fh, err := os.Open(filename)
		if err != nil {
			continue
		}
		fi, err := fh.Stat()
		if err != nil || fi.Size() == 0 {
			continue
		}

		sh := bufio.NewScanner(fh)
		for sh.Scan() {
			count, err := fmt.Sscan(sh.Text(), &ip, &e.last, &e.expire, &e.count)
			if err == nil && count == 4 {
				db[ip] = e
			}
		}
		_ = fh.Close()
		log.Printf("readDB: loaded %d entries", len(db))
		break
	}
}
