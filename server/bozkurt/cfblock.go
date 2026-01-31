package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    //"os"
)

// IPRule Cloudflare API için JSON payload
type IPRule struct {
    Mode        string `json:"mode"`        // "block", "challenge", "whitelist"
    Configuration struct {
        Target string `json:"target"` // "ip"
        Value  string `json:"value"`  // IP adresi
    } `json:"configuration"`
    Notes string `json:"notes"`
}

// Block fonksiyonu
func Block(ips map[string]string, zoneID string, apiToken string) {
    url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules", zoneID)

    client := &http.Client{}

    for ip, note := range ips {
        rule := IPRule{
            Mode: "block",
            Notes: note,
        }
        rule.Configuration.Target = "ip"
        rule.Configuration.Value = ip

        payload, err := json.Marshal(rule)
        if err != nil {
            fmt.Printf("JSON hatası: %v\n", err)
            continue
        }

        req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
        if err != nil {
            fmt.Printf("Request hatası: %v\n", err)
            continue
        }

        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+apiToken)

        resp, err := client.Do(req)
        if err != nil {
            fmt.Printf("API hatası: %v\n", err)
            continue
        }

        defer resp.Body.Close()

        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            fmt.Printf("IP %s başarıyla engellendi\n", ip)
        } else {
            fmt.Printf("IP %s engellenemedi. Status: %d\n", ip, resp.StatusCode)
        }
    }
}

func main() {
    
    ips := map[string]string{
        "193.233.254.7": "DDoS tespit edildi",
        "174.138.61.184": "Brute force attack",
    }

    zoneID   := "a44f2e6a7177086ed0013b06d968e370"     // Cloudflare zone ID
    apiToken := "1wBfVzNLSzrKEFSDlPxYyMyq7c4BMsudfe59Qz4t" // Cloudflare API Token

    Block(ips, zoneID, apiToken)
}
