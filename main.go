package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/zenthangplus/goccm"
)

var (
	invalid     int
	valid       int
	errors      int
	checked     int
	finished    bool
	not_enought int
)

type InviteCode struct {
	Code      string      `json:"code"`
	Type      int         `json:"type"`
	ExpiresAt interface{} `json:"expires_at"`
	Guild     struct {
		ID                string      `json:"id"`
		Name              string      `json:"name"`
		Splash            string      `json:"splash"`
		Banner            string      `json:"banner"`
		Description       interface{} `json:"description"`
		Icon              string      `json:"icon"`
		Features          []string    `json:"features"`
		VerificationLevel int         `json:"verification_level"`
		VanityURLCode     string      `json:"vanity_url_code"`
		Nsfw              bool        `json:"nsfw"`
		NsfwLevel         int         `json:"nsfw_level"`
	} `json:"guild"`
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"channel"`
	ApproximateMemberCount   int `json:"approximate_member_count"`
	ApproximatePresenceCount int `json:"approximate_presence_count"`
}

type Config struct {
	Threads       int    `json:"threads"`
	MinPercentage int    `json:"min_percentage"`
	MinMembers    int    `json:"min_members"`
	ProxiesType   string `json:"proxies_type"`
	ProxiesPath   string `json:"proxies_path"`
}

func changeTerminalName(name string) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "title", name)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd := exec.Command("bash", "-c", "echo -n -e '\033]0;"+name+"\007'")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	defer file.Close()
	return lines, scanner.Err()
}

func checkInvite(invite string, proxy string, config Config) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://canary.discord.com/api/v6/invite/%s?with_counts=true", invite), nil)
	if err != nil {
		fmt.Printf("[ERROR] %s", err)
		errors++
		return
	}
	
	url_i := url.URL{}
	url_proxy, _ := url_i.Parse(proxy)

	transport := http.Transport{}
	transport.Proxy = http.ProxyURL(url_proxy)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{Transport: &transport}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] %s", err)
		errors++
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[ERROR] %s", err)
		errors++
		return
	}
	
	var code = InviteCode{}
	err = json.Unmarshal(body, &code)
	if err != nil {
		fmt.Printf("Invalid invite: %s\n", invite)
		invalid++
		return
	}
	
	percentage := float64(code.ApproximatePresenceCount) / float64(code.ApproximateMemberCount) * 100
	if code.ApproximateMemberCount < config.MinMembers {
		fmt.Printf("%s - Not enough members\n", code.Code)
		not_enought++
		return
	}
	if percentage < float64(config.MinPercentage) {
		fmt.Printf("%s - Not enough online members\n", code.Code)
		not_enought++
		return
	}

	fmt.Printf("%s - Online: %d/%d (%.2f%%)\n", code.Code, code.ApproximatePresenceCount, code.ApproximateMemberCount, percentage)
	file, err := os.OpenFile("code.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
		errors++
	}
	
	_, err = file.WriteString(fmt.Sprintf("%s:%s\n", code.Code, code.Guild.ID))
	if err != nil {
		log.Fatal(err)
		errors++
	} else {
		valid++
	}

	file.Close()
}

func loadConfig() Config {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	
	var config = Config{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		err = json.Unmarshal([]byte(scanner.Text()), &config)
		if err != nil {
			log.Fatal(err)
		}
	}

	return config
}

func include_codes(invite string, list []string, element int) bool {
	for _, value := range list {
		if value == strings.Split(invite, ":")[element] {
			return true
		}
	}

	return false
}

func main() {
	config := loadConfig()

	code, err := readLines("code.txt")
	lines, err := readLines("invites.txt")
	proxies, err := readLines(config.ProxiesPath)

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if finished {
				break
			}

			time.Sleep(time.Second * 1)
			changeTerminalName(fmt.Sprintf("Checked: %d/%d - Invalid: %d - Error: %d - Valid: %d - NotEnought: %d", checked, len(lines), invalid, errors, valid, not_enought))
		}
	}()

	c := goccm.New(config.Threads)
	for _, invite := range lines {
		c.Wait()
		go func(invite string) {
			if !include_codes(invite, code, 0) {
				checkInvite(invite, config.ProxiesType+"://"+proxies[rand.Intn(len(proxies))], config)
			} else {
				fmt.Printf("Duplicate invite: %s\n", invite)
			}
			checked++
			c.Done()
		}(invite)
	}

	c.WaitAllDone()
	finished = true
}