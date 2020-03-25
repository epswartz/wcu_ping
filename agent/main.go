package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	ping "github.com/sparrc/go-ping"
)

const (
	CONFIG_LOC = "https://raw.githubusercontent.com/epswartz/wcu_ping/master/config.json" // Default location of config
)

// Config holds the json cfg from github
type Config struct {
	Count   int    `json:"count"`
	Address string `json:"address"`
	SendLoc string `json:"sendLoc"`
}

func getConfig(configAddress string) (Config, error) {
	defaultConfig := Config{
		Count:   10,
		Address: "8.8.8.8",
		SendLoc: "https://enyarz9q8ianh.x.pipedream.net", //FIXME swap for something else not requestbin
	}
	resp, err := http.Get(configAddress)
	if err != nil {
		return defaultConfig, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return defaultConfig, err
	}

	var cfg Config
	err = json.Unmarshal(body, &cfg)
	if err != nil {
		return defaultConfig, err
	}
	return cfg, nil
}

func main() {

	// Reach out to github and read the config
	cfg, err := getConfig(CONFIG_LOC)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(cfg)

	pinger, err := ping.NewPinger(cfg.Address)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	pinger.Count = cfg.Count // It does 10 packets

	headerLine := "seq,timestamp,rtt,address"
	pingLines := []string{headerLine}
	// On recieve, put data into graph
	pinger.OnRecv = func(pkt *ping.Packet) {
		lineParts := []string{
			strconv.Itoa(pkt.Seq),
			strconv.FormatInt(time.Now().Unix(), 10),
			pkt.Rtt.String(),
			pkt.Addr,
		}
		pingLines = append(pingLines, strings.Join(lineParts, ","))
	}

	pinger.Run() // blocks until it's done

	for _, l := range pingLines {
		fmt.Println(l)
	}

	// Send off the response
	var jsonStr = []byte(strings.Join(pingLines, "\n"))
	req, err := http.NewRequest("POST", cfg.SendLoc, bytes.NewBuffer(jsonStr))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "text/csv")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}
