package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const rounds = 10
const probesPerRound = 1
const maxHops = 30
const roundDelay = 1500 * time.Millisecond

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
			Dst   string `json:"dst"`
			Tests int    `json:"tests"`
			Psize int    `json:"psize"`
		} `json:"mtr"`
		Hubs []hop `json:"hubs"`
	} `json:"report"`
}

// parsedHop holds intermediate data during traceroute output parsing
type parsedHop struct {
	num        int
	host       string
	rttSamples []float64
	timeouts   int
}

func runMtr(target string) (string, error) {
	fmt.Printf("正在追蹤路由到 %s（共 %d 輪）...\n", target, rounds)

	hopMap := make(map[int]*parsedHop)
	var hopOrder []int

	for r := 1; r <= rounds; r++ {
		fmt.Printf("  第 %d/%d 輪...\n", r, rounds)
		out, err := runTraceroute(target)
		if err != nil && len(out) == 0 {
			if r == 1 {
				return "", fmt.Errorf("traceroute 執行失敗：%w", err)
			}
			continue
		}
		mergeRound(hopMap, &hopOrder, string(out))
		if r < rounds {
			time.Sleep(roundDelay)
		}
	}

	if len(hopOrder) == 0 {
		return "", fmt.Errorf("無法解析 traceroute 輸出")
	}

	hops := finalizeHops(hopMap, hopOrder)
	hops = truncateAtTarget(hops, target)

	var report mtrReport
	report.Report.Mtr.Dst = target
	report.Report.Mtr.Tests = rounds
	report.Report.Mtr.Psize = 40
	report.Report.Hubs = hops

	result, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	fmt.Println("完成！")
	return string(result), nil
}

func runTraceroute(target string) ([]byte, error) {
	if runtime.GOOS == "windows" {
		return exec.Command("tracert", "-h", strconv.Itoa(maxHops), "-w", "1000", target).Output()
	}
	return exec.Command("traceroute",
		"-n",
		"-m", strconv.Itoa(maxHops),
		"-q", strconv.Itoa(probesPerRound),
		"-w", "2",
		target,
	).Output()
}

var newHopRegex = regexp.MustCompile(`^\s*(\d+)\s+(.+)$`)
var contLineRegex = regexp.MustCompile(`^\s{4,}(\S.+)$`)
var rttRegex = regexp.MustCompile(`([\d.]+)\s*ms`)
var ipRegex = regexp.MustCompile(`\(?([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})\)?`)

// mergeRound parses one round of traceroute output and merges samples into hopMap
func mergeRound(hopMap map[int]*parsedHop, hopOrder *[]int, output string) {
	lines := strings.Split(output, "\n")
	currentHop := 0

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if m := newHopRegex.FindStringSubmatch(line); m != nil {
			hopNum, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			rest := m[2]
			if strings.Contains(rest, "hops max") {
				continue
			}

			p, exists := hopMap[hopNum]
			if !exists {
				p = &parsedHop{num: hopNum}
				hopMap[hopNum] = p
				*hopOrder = append(*hopOrder, hopNum)
			}
			if p.host == "" {
				if ipM := ipRegex.FindStringSubmatch(rest); ipM != nil {
					p.host = ipM[1]
				}
			}
			collectLine(p, rest)
			currentHop = hopNum
			continue
		}

		if currentHop > 0 {
			if m := contLineRegex.FindStringSubmatch(line); m != nil {
				if p, ok := hopMap[currentHop]; ok {
					collectLine(p, m[1])
				}
			}
		}
	}
}

func parseTraceroute(output string) []hop {
	hopMap := make(map[int]*parsedHop)
	var hopOrder []int
	mergeRound(hopMap, &hopOrder, output)
	return finalizeHops(hopMap, hopOrder)
}

func finalizeHops(hopMap map[int]*parsedHop, hopOrder []int) []hop {
	result := make([]hop, 0, len(hopOrder))
	for _, num := range hopOrder {
		p := hopMap[num]
		received := len(p.rttSamples)
		total := received + p.timeouts
		if total == 0 {
			total = rounds
		}

		host := p.host
		if host == "" {
			host = "???"
		}

		if received == 0 {
			result = append(result, hop{Count: p.num, Host: host, Loss: 100, Snt: total})
			continue
		}

		var sum, best, worst float64
		best = p.rttSamples[0]
		worst = p.rttSamples[0]
		for _, v := range p.rttSamples {
			sum += v
			if v < best {
				best = v
			}
			if v > worst {
				worst = v
			}
		}
		avg := sum / float64(received)

		var variance float64
		for _, v := range p.rttSamples {
			d := v - avg
			variance += d * d
		}
		stddev := 0.0
		if received > 1 {
			stddev = sqrtF(variance / float64(received))
		}

		loss := float64(total-received) / float64(total) * 100

		result = append(result, hop{
			Count: p.num,
			Host:  host,
			Loss:  roundMs(loss),
			Snt:   total,
			Avg:   roundMs(avg),
			Best:  roundMs(best),
			Wrst:  roundMs(worst),
			StDev: roundMs(stddev),
		})
	}
	return result
}

func collectLine(p *parsedHop, text string) {
	for _, r := range rttRegex.FindAllStringSubmatch(text, -1) {
		v, err := strconv.ParseFloat(r[1], 64)
		if err == nil {
			p.rttSamples = append(p.rttSamples, v)
		}
	}
	p.timeouts += strings.Count(text, "*")
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

// truncateAtTarget cuts hops at the first one whose host matches the resolved target IP.
// Some traceroute implementations show duplicate trailing hops (with bogus loss) after
// reaching the destination — we drop them.
func truncateAtTarget(hops []hop, target string) []hop {
	targetIP := resolveTarget(target)
	if targetIP == "" {
		return hops
	}
	for i, h := range hops {
		if h.Host == targetIP {
			return hops[:i+1]
		}
	}
	return hops
}

func resolveTarget(target string) string {
	if ip := net.ParseIP(target); ip != nil {
		return target
	}
	addrs, err := net.LookupHost(target)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	return addrs[0]
}
