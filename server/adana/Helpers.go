package main

import(
	"strconv"
	"strings"
	"log"
	"net/url"
	"net"
)

func Ranges(s string) []string {
	s = strings.TrimSpace(s)
	if _, err := strconv.Atoi(s); err == nil {
		return []string{s}
	}

	var a, b int
	if strings.Contains(s, "-") {
		p := strings.SplitN(s, "-", 2)
		ai, e1 := strconv.Atoi(p[0])
		bi, e2 := strconv.Atoi(p[1])
		if e1 != nil || e2 != nil { return []string{} }
		a, b = ai, bi
	} else if strings.Contains(s, "+") {
		p := strings.SplitN(s, "+", 2)
		ai, e1 := strconv.Atoi(p[0])
		n, e2 := strconv.Atoi(p[1])
		if e1 != nil || e2 != nil { return []string{} }
		a, b = ai, ai+n
	} else {
		return []string{}
	}
	if a > b { return []string{} }
	out := make([]string, 0, b-a+1)
	for i := a; i <= b; i++ {
		out = append(out, strconv.Itoa(i))
	}
	return out
}



func BalancerHash(url string, r int) int {
	if r <= 0 {
		return 0
	}
	var hash uint32
	for i := 0; i < len(url); i++ {
		hash = hash*31 + uint32(url[i])
	}
	return int(hash % uint32(r))
}



func HostDomain(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return h
}

func UrlPath(s string) string {
	i := strings.Index(s, "/")
	if i == -1 {
		return ""
	}
	p := s[i+1:]
	return strings.TrimPrefix(p, "/")
}

func UrlDomain(hostport string) string {
	if i := strings.Index(hostport, "/"); i != -1 {
		hostport = hostport[:i]
	}
	return hostport
}

func ParseURLS(list []string) []*url.URL {
	out := make([]*url.URL, 0, len(list))
	for _, s := range list {
		u, err := url.Parse(s)
		if err != nil {
			if !strings.Contains(s, "://") {
				if u2, err2 := url.Parse("http://" + s); err2 == nil {
					u = u2
				} else {
					log.Printf("bad backend url %q: %v", s, err)
					continue
				}
			} else {
				log.Printf("bad backend url %q: %v", s, err)
				continue
			}
		}
		out = append(out, u)
	}
	return out
}

