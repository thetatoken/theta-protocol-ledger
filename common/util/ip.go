package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// GetPublicIP returns the public IP address of the current node
func GetPublicIP() (string, error) {
	ipMap := make(map[string]int)
	numSources := 3
	wait := &sync.WaitGroup{}
	mu := &sync.RWMutex{}

	go func() {
		resp, err := http.Get("http://myexternalip.com/raw")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					ip := strings.TrimSpace(string(bodyBytes))
					mu.Lock()
					defer mu.Unlock()
					ipMap[ip]++
				}
			}
		}
		wait.Done()
	}()
	wait.Add(1)

	go func() {
		resp, err := http.Get("http://whatismyip.akamai.com")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					ip := strings.TrimSpace(string(bodyBytes))
					mu.Lock()
					defer mu.Unlock()
					ipMap[ip]++
				}
			}
		}
		wait.Done()
	}()
	wait.Add(1)

	var defaultIP string
	go func() {
		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd", "/c", "nslookup myip.opendns.com resolver1.opendns.com")
			out, err := cmd.CombinedOutput()
			if err == nil {
				res := strings.TrimSpace(string(out))
				ip := res[strings.LastIndex(res, " ")+1:]
				defaultIP = ip
				mu.Lock()
				defer mu.Unlock()
				ipMap[ip]++
			}
		} else {
			cmd := exec.Command("bash", "-c", "dig @resolver1.opendns.com ANY myip.opendns.com +short")
			out, err := cmd.CombinedOutput()
			if err == nil {
				ip := strings.TrimSpace(string(out))
				defaultIP = ip
				mu.Lock()
				defer mu.Unlock()
				ipMap[ip]++
			}
		}
		wait.Done()
	}()
	wait.Add(1)

	wait.Wait()

	var majorityIP string
	for ip, cnt := range ipMap {
		if cnt > numSources/2 {
			majorityIP = ip
			break
		}
	}

	if majorityIP != "" {
		return majorityIP, nil
	}

	if defaultIP != "" {
		return defaultIP, nil
	}

	return "", fmt.Errorf("Can't get external IP")
}
