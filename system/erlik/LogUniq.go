package main

import (
	"fmt"
	"math"
	"os"
	"time"

  "github.com/cilium/ebpf"
)

const (
	numBuckets  = 4096
	alphaM      = 0.721347 // 4096 kova için sabit HLL katsayısı
	hllMapPath  = "/sys/fs/bpf/map_HLL_TABLE"
	statMapPath = "/sys/fs/bpf/map_REPORT"
	
)

type Report struct {
	Pass  uint32
	Drop  uint32
	Byte  uint64
	Uniq  uint32
	_     [4]byte
}

func LogUniq() error {
	// 1. Sabitlenmiş (pinned) XDP haritasını yükle
	hllMap, err := ebpf.LoadPinnedMap(hllMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
	defer hllMap.Close()

	statMap, err := ebpf.LoadPinnedMap(statMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
	defer statMap.Close()


	numCPU, err := ebpf.PossibleCPU()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] ebpf.PossibleCPU() %v\n", err)
		return err
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Hesaplama ve Sıfırlama fonksiyonunu çağır
		count, err := calculateAndFlushHLL(hllMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[X] Hesaplama hatası: %v\n", err)
			continue
		}
		
		fmt.Printf("Time=%s Uniq=%d\n", time.Now().Format("15:04:05"), count)

		now    := time.Now()
		period := uint32(now.Hour()*60 + now.Minute())


		key := uint32(period)
		reports := make([]Report, numCPU)
		if err := statMap.Lookup(&key, &reports); err != nil {
			fmt.Fprintf(os.Stderr, "[X] key not found in REPORT %v\n", err)
			return err
		}

		reports[0].Uniq = uint32(count)
		if err := statMap.Update(&key, reports, ebpf.UpdateAny); err != nil {
			fmt.Fprintf(os.Stderr, "[X] Error when update REPORT %v\n", err)
			return err
		}



	}
	return nil
}

func calculateAndFlushHLL(m *ebpf.Map) (int64, error) {
	var sumInverse float64 = 0
	var zeroBuckets float64 = 0

	// Kovaları haritadan oku
	for i := uint32(0); i < numBuckets; i++ {
		var val uint8
		if err := m.Lookup(i, &val); err != nil {
			return 0, fmt.Errorf("kova %d okunamadı: %w", i, err)
		}

		if val == 0 {
			zeroBuckets++
		}

		// Harmonik ortalama için 2^(-val) toplamı
		sumInverse += math.Pow(2, -float64(val))
	}

	// Standart HyperLogLog ham tahmini (Raw Estimate)
	estimate := alphaM * float64(numBuckets) * float64(numBuckets) / sumInverse

	// Küçük veri kümeleri için düzeltme (Linear Counting)
	if estimate <= 2.5*float64(numBuckets) {
		if zeroBuckets > 0 {
			estimate = float64(numBuckets) * math.Log(float64(numBuckets)/zeroBuckets)
		}
	}

	// Bir sonraki saat için haritadaki tüm kovaları sıfırla
	zeroVal := uint8(0)
	for i := uint32(0); i < numBuckets; i++ {
		if err := m.Update(i, &zeroVal, ebpf.UpdateExist); err != nil {
			return 0, fmt.Errorf("kova %d sıfırlanamadı: %w", i, err)
		}
	}

	return int64(math.Round(estimate)), nil
}
