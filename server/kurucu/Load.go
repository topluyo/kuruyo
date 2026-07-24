package main

import (
	"encoding/json"
  "os"
  "strings"
  "strconv"
)

type Config struct {
	IP       string           `json:"ip"`
	HTTP     int              `json:"http"`
	HTTPS    int              `json:"https"`
	Engine   string           `json:"engine"`
	RateSize int              `json:"rateSIZE"`
	Cert     string           `json:"cert"`
	Priv     string           `json:"priv"`
	Routes   map[string]*Route `json:"routes"`
}

type Route struct {
  Name        string
	Description string   `json:"description"`
	Ports       string   `json:"ports"`
	Serve       string   `json:"serve"`
	Levels      []string `json:"levels"`
}

func Load(file string) *Config {
	jsonData, err := os.ReadFile(file)
	if err != nil {
		write("[X] configuration file reading error:",err)
    return nil
	}

	var config Config

	err = json.Unmarshal(jsonData, &config)
	if err != nil {
    write("[X] configuration json reading error:",err)
		return nil
	}

	return &config
}

func LoadRoute(conf *Config, port int) *Route {
	portString := ToString(port)
	var founded *Route
	for i := range conf.Routes {
		r := conf.Routes[i]
    r.Name = i
		for _, p := range Ranges(r.Ports) {
			if p == portString {
				if founded == nil {
					founded = r
				}
				if founded != nil && founded.Serve == "" {
					founded = r
				}
			}
		}
	}
	return founded
}



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