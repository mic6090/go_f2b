package main

import (
	"errors"
	"fmt"
	"github.com/pelletier/go-toml"
	"os"
	"regexp"
	"strings"
)

const MAINSECTION = "main"
const DUMPNAME = "dump"
const DUMPCOUNT = 5

type mainConfig struct {
	BanTime        int
	MaxRetry       int
	DBPurgeAge     int64
	DBDumpPath     string
	globalIgnoreIP []IPNet
	services       []serviceConfig
}

type serviceConfig struct {
	service         string
	logName         string
	regexps         []*regexp.Regexp
	serviceIgnoreIP []IPNet
}

var Conf mainConfig

func readConfig(filename string) error {
	config, err := toml.LoadFile(filename)
	if err != nil {
		return err
	}

	mainSection, ok := config.Get(MAINSECTION).(*toml.Tree)
	if !ok {
		return errors.New("section " + MAINSECTION + " not found in config file")
	}
	if err = mainSection.Unmarshal(&Conf); err != nil {
		return err
	}
	if Conf.globalIgnoreIP, err = getIgnoreIP(mainSection); err != nil {
		return err
	}
	if Conf.MaxRetry < 1 {
		return errors.New("invalid maxretry value")
	}

	fi, err := os.Stat(Conf.DBDumpPath)
	if err != nil {
		if err = os.MkdirAll(Conf.DBDumpPath, 0755); err != nil {
			return err
		}
	} else {
		if !fi.IsDir() {
			return errors.New("db dump path set to file")
		}
	}

	var reString string
	for _, service := range config.Keys() {
		if service == MAINSECTION {
			continue
		}
		section := config.Get(service).(*toml.Tree)
		srv := serviceConfig{service: service}
		for k, v := range section.ToMap() {
			switch {
			case k == "file":
				srv.logName, ok = v.(string)
				if !ok {
					return fmt.Errorf("bad value for \"%s\" parameter in section \"%s\"", k, service)
				}
			case strings.HasPrefix(k, "regex"):
				reString, ok = v.(string)
				if !ok {
					return fmt.Errorf("bad value for \"%s\" parameter in section \"%s\"", k, service)
				}
				re, err := regexp.Compile(reString)
				if err != nil {
					return fmt.Errorf("bad regexp in section \"%s\": parameter \"%s\", error \"%s\"", service, k, err)
				}
				srv.regexps = append(srv.regexps, re)
			}
		}
		if len(srv.regexps) == 0 {
			return fmt.Errorf("no regexps for service \"%s\"", service)
		}
		if srv.serviceIgnoreIP, err = getIgnoreIP(section); err != nil {
			return err
		}
		Conf.services = append(Conf.services, srv)
	}
	return nil
}

func parseIPList(list string) ([]IPNet, error) {
	var res []IPNet

	for _, addr := range strings.Split(strings.ReplaceAll(list, ",", " "), " ") {
		if addr == "" {
			continue
		}
		_, ipnet, err := ParseCIDR(addr)
		if err != nil {
			return nil, err
		}
		res = append(res, ipnet)
	}
	return res, nil
}

func getIgnoreIP(tree *toml.Tree) ([]IPNet, error) {
	if !tree.Has("ignoreip") {
		return nil, nil
	}
	ips, ok := tree.Get("ignoreip").(string)
	if !ok {
		return nil, errors.New("wrong \"ignoreip\" parameter")
	}
	return parseIPList(ips)
}
