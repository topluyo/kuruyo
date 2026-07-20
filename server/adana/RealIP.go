package main

import(
  "net"
  "net/http"
  "strings"
  "context"
)

var CloudflareIPS = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

var CloudflareNets []*net.IPNet

func Init_CLOUDFLARENETS() {
	CloudflareNets = make([]*net.IPNet, 0, len(CloudflareIPS))
	for _, cidr := range CloudflareIPS {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		CloudflareNets = append(CloudflareNets, n)
	}
}

func IsCloudflareFast(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, n := range CloudflareNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func CalcRealIP(r *http.Request) string {
	ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	ip := net.ParseIP(ipStr)
	if !IsCloudflareFast(ip) {
		return ipStr
	}
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i != -1 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	return ipStr
}

type CTX_KEY_REAL_IP struct{}

//@ SetIP -> run
func SetIP(r *http.Request) (*http.Request, string) {
	ip := CalcRealIP(r)
	ctx := context.WithValue( r.Context(), CTX_KEY_REAL_IP{}, ip)
	return r.WithContext(ctx), ip
}

//@ GetIP -> run
func GetIP(r *http.Request) string {
	if ip, ok := r.Context().Value(CTX_KEY_REAL_IP{}).(string); ok {
		return ip
	}
	return ""
}


