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
	blacklisted int
	higthlevel  int
	blacklist   []string
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
	BlacklistedName []string `json:"blacklist_word"`
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

func Include(item string, list []string) bool {
	for _, value := range list {
		if value == item {
			return true
		}
	}

	return false
}

func checkInvite(invite string, proxy string, config Config) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://canary.discord.com/api/v6/invite/%s?with_counts=true", invite), nil)
	if err != nil {
		//fmt.Printf("[ERROR] %s", err)
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
		//fmt.Printf("[ERROR] %s", err)
		errors++
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("[ERROR] %s", err)
		errors++
		return
	}

	cfile, err := os.OpenFile("checked.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		//log.Fatal(err)
		errors++
	}

	_, err = cfile.WriteString(fmt.Sprintf("%s\n", invite))
	if err != nil {
		//log.Fatal(err)
		errors++
	} else {
		valid++
	}

	cfile.Close()

	var code = InviteCode{}
	err = json.Unmarshal(body, &code)
	if err != nil {
		//fmt.Printf("Invalid invite: %s\n", invite)
		invalid++
		return
	}

	if Include(code.Guild.ID, blacklist) {
		//fmt.Printf("Blacklisted invite: %s\n", invite)
		blacklisted++
		return
	}

	for _, word := range config.BlacklistedName {
		if strings.Contains(code.Guild.Name, word) {
			//fmt.Printf("Blacklisted word: %s\n", invite)
			blacklisted++
			return
		}
	}

	// already checked uwu
	blacklist = append(blacklist, code.Guild.ID)

	// check if need to get phone verfied with the verification level
	if code.Guild.VerificationLevel == 4 {
		higthlevel++
		return
	}

	percentage := float64(code.ApproximatePresenceCount) / float64(code.ApproximateMemberCount) * 100
	if code.ApproximateMemberCount < config.MinMembers {
		//fmt.Printf("%s - Not enough members\n", code.Code)
		not_enought++
		return
	}
	if percentage < float64(config.MinPercentage) {
		//fmt.Printf("%s - Not enough online members\n", code.Code)
		not_enought++
		return
	}

	fmt.Printf("%s - Online: %d/%d (%.2f%%)\n", code.Code, code.ApproximatePresenceCount, code.ApproximateMemberCount, percentage)
	file, err := os.OpenFile("code.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		//log.Fatal(err)
		errors++
	}

	_, err = file.WriteString(fmt.Sprintf("%s:%s\n", code.Code, code.Guild.ID))
	if err != nil {
		//log.Fatal(err)
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

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		item = strings.ToLower(item) // lel lowercase

		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func main() {
	config := loadConfig()

	code, err := readLines("code.txt")
	lines, err := readLines("invites.txt")
	proxies, err := readLines(config.ProxiesPath)
	blacklist, err = readLines("blacklist.txt")
	checked_codes, err := readLines("checked.txt")

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if finished {
				break
			}

			time.Sleep(time.Second * 1)
			changeTerminalName(fmt.Sprintf("Checked: %d/%d - Invalid: %d - Error: %d - Valid: %d - NotEnought: %d - Blacklisted: %d VerificationLevel: %d", checked, len(lines), invalid, errors, valid, not_enought, blacklisted, higthlevel))
		}

		fmt.Printf("\n\nChecked: %d/%d - Invalid: %d - Error: %d - Valid: %d - NotEnought: %d - Blacklisted: %d VerificationLevel: %d", checked, len(lines), invalid, errors, valid, not_enought, blacklisted, higthlevel)
	}()

	// blacklist already checked guild
	for _, invite := range code {
		blacklist = append(blacklist, strings.Split(invite, ":")[1])
	}

	// blacklist already checked invites
	for _, invite := range checked_codes {
		code = append(code, strings.Split(invite, ":")[0])
	}

	c := goccm.New(config.Threads)
	for _, invite := range lines {
		c.Wait()
		go func(invite string) {
			if strings.Contains(invite, ".gg/") {
				invite = strings.Split(invite, ".gg/")[1]
			}

			if !include_codes(invite, code, 0) {
				checkInvite(invite, config.ProxiesType+"://"+proxies[rand.Intn(len(proxies))], config)
			} else {
				//fmt.Printf("Duplicate invite: %s\n", invite)
				blacklisted++
			}
			checked++
			c.Done()
		}(invite)
	}

	c.WaitAllDone()
	finished = true
}
