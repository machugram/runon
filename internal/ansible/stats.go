package ansible

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type HostStats struct {
	Changed     int
	Failures    int
	Unreachable int
}

// AssertClean fails when any host changed, failed, or was unreachable.
func AssertClean(stats map[string]HostStats) error {
	if len(stats) == 0 {
		return fmt.Errorf("ansible stats were empty")
	}
	hosts := make([]string, 0, len(stats))
	for host := range stats {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	var problems []string
	for _, host := range hosts {
		stat := stats[host]
		if stat.Failures > 0 || stat.Unreachable > 0 {
			problems = append(problems, fmt.Sprintf("%s failures=%d unreachable=%d", host, stat.Failures, stat.Unreachable))
		}
		if stat.Changed > 0 {
			problems = append(problems, fmt.Sprintf("%s changed=%d", host, stat.Changed))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("playbook was not idempotent: %s", strings.Join(problems, "; "))
	}
	return nil
}

var recapLine = regexp.MustCompile(`^(\S+)\s+:\s+ok=\d+\s+changed=(\d+)\s+unreachable=(\d+)\s+failed=(\d+)`)

// ParseRecap reads the PLAY RECAP block from Ansible's default callback.
func ParseRecap(stdout []byte) (map[string]HostStats, error) {
	stats := map[string]HostStats{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		line := stripANSI(scanner.Text())
		match := recapLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		changed, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, err
		}
		unreachable, err := strconv.Atoi(match[3])
		if err != nil {
			return nil, err
		}
		failed, err := strconv.Atoi(match[4])
		if err != nil {
			return nil, err
		}
		stats[match[1]] = HostStats{Changed: changed, Failures: failed, Unreachable: unreachable}
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("ansible output did not contain a play recap")
	}
	return stats, nil
}

func stripANSI(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] != 0x1b {
			b.WriteByte(line[i])
			continue
		}
		for i < len(line) && line[i] != 'm' {
			i++
		}
	}
	return b.String()
}
