package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cycles = 10
const maxHops = 30

type hop struct {
	Count int     `json:"count"`
	Host  string  `json:"host"`
	Loss  float64 `json:"Loss%"`
	Snt   int     `json:"Snt"`
	Avg   float64 `json:"Avg"`
	Best  float64 `json:"Best"`
	Wrst  float64 `json:"Wrst"`
	StDev float64 `json:"StDev"`
}

type mtrReport struct {
	Report struct {
		Mtr struct {
			Dst        string `json:"dst"`
			Src        string `json:"src"`
			Tos        int    `json:"tos"`
			Psize      int    `json:"psize"`
			Bitpattern int    `json:"bitpattern"`
			Tests      int    `json:"tests"`
		} `json:"mtr"`
		Hubs []hop `json:"hubs"`
	} `json:"report"`
}

func runMtr(target string) (string, error) {
	fmt.Printf("正在解析 %s ...\n", target)
	destIP, err := resolveHost(target)
	if err != nil {
		return "", fmt.Errorf("無法解析主機名稱：%w", err)
	}
	fmt.Printf("目標：%s (%s)\n", target, destIP)

	hops, err := tracePath(destIP, target)
	if err != nil {
		return "", err
	}

	var report mtrReport
	report.Report.Mtr.Dst = target
	report.Report.Mtr.Tests = cycles
	report.Report.Mtr.Psize = 64
	report.Report.Hubs = hops

	out, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func resolveHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	return addrs[0], nil
}

func tracePath(destIP, target string) ([]hop, error) {
	var hops []hop
	consecutiveUnknown := 0

	for ttl := 1; ttl <= maxHops; ttl++ {
		fmt.Printf("  測試第 %d 跳...\r", ttl)
		h := probeHop(ttl, target, destIP)
		hops = append(hops, h)

		if h.Host == "???" {
			consecutiveUnknown++
		} else {
			consecutiveUnknown = 0
		}

		// 到達目的地
		if h.Host == destIP {
			break
		}
		// 連續 5 個 ??? 停止
		if consecutiveUnknown >= 5 {
			break
		}
	}
	fmt.Println("  完成！                    ")
	return hops, nil
}

func probeHop(ttl int, target, destIP string) hop {
	rtts, hopIP := pingTTLMulti(ttl, cycles, target)

	sent := cycles
	received := 0
	var sum, best, worst float64
	best = 999999

	for _, rtt := range rtts {
		if rtt < 0 {
			continue
		}
		received++
		sum += rtt
		if rtt < best {
			best = rtt
		}
		if rtt > worst {
			worst = rtt
		}
	}

	loss := float64(sent-received) / float64(sent) * 100

	if received == 0 || hopIP == "" {
		return hop{Count: ttl, Host: "???", Loss: 100, Snt: sent}
	}

	avg := sum / float64(received)

	var variance float64
	for _, rtt := range rtts {
		if rtt >= 0 {
			d := rtt - avg
			variance += d * d
		}
	}
	stddev := 0.0
	if received > 1 {
		stddev = sqrtF(variance / float64(received))
	}

	return hop{
		Count: ttl,
		Host:  hopIP,
		Loss:  loss,
		Snt:   sent,
		Avg:   roundMs(avg),
		Best:  roundMs(best),
		Wrst:  roundMs(worst),
		StDev: roundMs(stddev),
	}
}

var rttRegex = regexp.MustCompile(`time[<=]([\d.]+)\s*ms`)
var hopIPRegex = regexp.MustCompile(`(?:From|from)\s+([\d.]+)`)

// pingTTLMulti 平行跑 n 次 ping，回傳所有 RTT 和這個 hop 的 IP
func pingTTLMulti(ttl, n int, target string) ([]float64, string) {
	type result struct {
		rtt float64
		ip  string
	}
	results := make([]result, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rtt, ip := singlePing(ttl, target)
			results[idx] = result{rtt, ip}
		}(i)
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	rtts := make([]float64, n)
	hopIP := ""
	for i, r := range results {
		rtts[i] = r.rtt
		if r.ip != "" && hopIP == "" {
			hopIP = r.ip
		}
	}
	return rtts, hopIP
}

func singlePing(ttl int, target string) (float64, string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-i", strconv.Itoa(ttl), "-w", "1000", target)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-m", strconv.Itoa(ttl), "-W", "1", target)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Run()

	output := out.String()

	// 取得回應 IP（TTL exceeded 的節點）
	ip := ""
	if m := hopIPRegex.FindStringSubmatch(output); m != nil {
		ip = m[1]
	} else if strings.Contains(output, target) && rttRegex.MatchString(output) {
		// 直接到達目的地
		ip = target
	}

	// 取得 RTT
	rtt := -1.0
	if m := rttRegex.FindStringSubmatch(output); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			rtt = v
		}
	}

	return rtt, ip
}

func sqrtF(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 50; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func roundMs(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
