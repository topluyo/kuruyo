package shield

import (
	"fmt"
	"log"
	"os/exec"
)

func banIPWithIPSet(ipsetName, ip string, banSeconds int) {
	if ipsetName == "" || ip == "" {
		return
	}

	args := []string{"add", ipsetName, ip, "-exist"}

	if banSeconds > 0 {
		args = append(args, "timeout", fmt.Sprintf("%d", banSeconds))
	}

	cmd := exec.Command("ipset", args...)
	if err := cmd.Run(); err != nil {
		log.Printf("[ipset] ban error %s -> %s: %v", ip, ipsetName, err)
	}
}
