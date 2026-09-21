package main

import (
	"log"
	"github.com/cilium/ebpf"
	"fmt"
	"time"
	"os"
	"math"
	"strings"
	//"sync"
	"sync/atomic"
)


type SyncObject struct {
	Period  uint32
	Time    uint32
	Mode    uint32
}

const (
	numBuckets  = 16384
	alphaM      = 0.721293
	hllMapPath  = "/sys/fs/bpf/map_HLL_TABLE"
	statMapPath = "/sys/fs/bpf/map_REPORT"
	programPath = "/sys/fs/bpf/bozkurt_program"	
)

type Report struct {
	Pass  uint32
	Drop  uint32
	Byte  uint64
	Uniq  uint32
	_     [4]byte
}

var TIME uint32
var PASS uint32
var DROP uint32
var BYTE uint64
var UNIQ uint32
var MODE uint32




var prog *ebpf.Program
var bssMap *ebpf.Map
var hllMap *ebpf.Map
var statMap *ebpf.Map
var numCPU int
var programID ebpf.ProgramID



func InitMaps() error{
	var err error

	prog, err = ebpf.LoadPinnedProgram(programPath,nil,)
  if err != nil {
		fmt.Println("[X] LoadPinnedProgram:", err)
		os.Exit(0)
		return err
  }
	info, err := prog.Info()
	if err != nil {
		log.Fatalf("Program bilgisi alınamadı: %v", err)
	}

	id, ok := info.ID()
	if !ok {
		return fmt.Errorf("program ID alınamadı")
	}
	programID = id


	mapIDs, available := info.MapIDs()
	if !available {
		log.Fatalf("Harita ID'leri mevcut değil")
	}
	// 3. Haritalar içinde tipi Array ve adı ".bss" olanı arayın
	for _, mID := range mapIDs {
		m, err := ebpf.NewMapFromID(mID)
		if err != nil {
			continue
		}
		mInfo, err := m.Info()
		if err == nil && mInfo.Name == ".bss" {
			bssMap = m
			break
		}
		m.Close()
	}
	if bssMap == nil {
		log.Fatalf(".bss haritası bulunamadı")
	}


	hllMap, err = ebpf.LoadPinnedMap(hllMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
	

	statMap, err = ebpf.LoadPinnedMap(statMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
	
	numCPU, err = ebpf.PossibleCPU()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] ebpf.PossibleCPU() %v\n", err)
		return err
	}
	return nil
}

func CloseMaps(){
	prog.Close()
	bssMap.Close()
	hllMap.Close()
	statMap.Close()
}

func IsProgramChanged() (bool, error) {
	currentProg, err := ebpf.LoadPinnedProgram(programPath, nil)
	if err != nil {
		return false, fmt.Errorf("pinned program yüklenemedi: %w", err)
	}
	defer currentProg.Close()

	info, err := currentProg.Info()
	if err != nil {
		return false, fmt.Errorf("program info alınamadı: %w", err)
	}

	id, ok := info.ID()
	if !ok {
		fmt.Errorf("program ID alınamadı")
		return false, fmt.Errorf("program ID alınamadı")
	}
	
	if id != programID {
		return true, nil
	}

	return false, nil
}

func ReInitMaps() error {
	// Yeni pinned programı aç
	newProg, err := ebpf.LoadPinnedProgram(programPath, nil)
	if err != nil {
		return fmt.Errorf("yeni pinned program yüklenemedi: %w", err)
	}

	newInfo, err := newProg.Info()
	if err != nil {
		newProg.Close()
		return fmt.Errorf("yeni program bilgisi alınamadı: %w", err)
	}

	// Yeni .bss map'i bul
	var newBSSMap *ebpf.Map
	mapIDs, available := newInfo.MapIDs()
	if !available {
		newProg.Close()
		return fmt.Errorf("yeni program için map ID'leri mevcut değil")
	}

	for _, mID := range mapIDs {
		m, err := ebpf.NewMapFromID(mID)
		if err != nil {
			continue
		}

		mInfo, err := m.Info()
		if err != nil {
			m.Close()
			continue
		}

		if mInfo.Name == ".bss" {
			newBSSMap = m
			break
		}

		m.Close()
	}

	if newBSSMap == nil {
		newProg.Close()
		return fmt.Errorf("yeni .bss haritası bulunamadı")
	}

	// Buraya kadar geldiysek yeni state geçerli.
	// Eski objeleri artık kapatabiliriz.
	if prog != nil {
		prog.Close()
	}

	if bssMap != nil {
		bssMap.Close()
	}

	// Yeni objeleri aktif hale getir
	prog = newProg
	bssMap = newBSSMap

	id, ok := newInfo.ID()
	if !ok {
		return fmt.Errorf("program ID alınamadı")
	}

	programID = id

	return nil
}


func UpdateTime(period uint32, time uint32) bool{
	var key uint32 = 0

	var data SyncObject
	if err := bssMap.Lookup(&key, &data); err != nil {
		log.Fatalf("[X] Veri okuma hatası: %v", err)
		return false
	}
	

	//now := time.Now()
	data.Period = period //uint32(now.Hour()*60 + now.Minute())
	data.Time   = time //uint32(now.Unix())
	data.Mode   = atomic.LoadUint32(&MODE)

	
	
	
  if err := bssMap.Update(&key, &data, ebpf.UpdateAny); err != nil {
		log.Fatalf("[X] veri güncellenemedi: %v", err)
		return false
  }else{
		//atomic.StoreUint64(&TIME, data.Period)
	}

	
	if err := bssMap.Lookup(&key, &data); err != nil {
		log.Fatalf("[X] Veri okuma hatası: %v", err)
	}
	
  return true
}





func UpdateUniq(period uint32, time uint32) error {
	
	// Hesaplama ve Sıfırlama fonksiyonunu çağır
	count, err := calculateAndFlushHLL(hllMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Hesaplama hatası: %v\n", err)
		return err
	}
	
	//fmt.Printf("Time=%s Uniq=%d\n", time.Now().Format("15:04:05"), count)
	//now    := time.Now()
	//period := uint32(now.Hour()*60 + now.Minute())


	key := period
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


	TIME = key
	PASS = 0
	DROP = 0
	BYTE = 0
	UNIQ = uint32(count)
	for i:=0 ; i<numCPU; i++{
		PASS += reports[i].Pass
		DROP += reports[i].Drop
		BYTE += reports[i].Byte
	}
	
	
	return nil
}

//@calculateAndFlushHLL
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



func MinuteLoop() {

	InitMaps()
	defer CloseMaps()

	now := time.Now()
	timer := time.NewTimer(time.Until(now.Truncate(time.Second).Add(time.Second)))
	<-timer.C

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		SecondLoop()
	}
}

func SecondLoop(){
	now := time.Now()
	period := uint32(now.Hour()*60 + now.Minute())
	ts := uint32(now.Unix())

	UpdateTime(period, ts)
	CheckIsClosed()
	if now.Second() == 0 {
		UpdateUniq( ((period-1)+1440)%1440 , ts)
		
		fmt.Printf("%02d:%02d %8d PASS %8d DROP %9d KB %8d IP %8d MODE\n",
			TIME/60, TIME%60, PASS, DROP, BYTE/1024, UNIQ, atomic.LoadUint32(&MODE) )
	}
}


func CreateMapServer(){
	go UnixSocketServer("/web/sockets/erlik.sock", func(request string) string {
		parameters := strings.Split(request, ",")
		

		if( request=="HELLO" ){
			return "HI!"
		}

		if( len(parameters) == 2 && parameters[0]=="MODE" ){
		  //	printf 'MODE,1\0' | socat - UNIX-CONNECT:/web/sockets/erlik.sock
			num := uint32(ToNumber(parameters[1]))
			atomic.StoreUint32(&MODE, num)
			SecondLoop()
			return "MODE:"+ToString(int(num))
		}

		if( len(parameters) == 3 && parameters[0]=="DROP" ){
			Add("block",parameters[1], uint64(ToNumber(parameters[2])) , false)
			return "BLOCKED:" + parameters[1] + ":" + parameters[2]
		}
		if( len(parameters) == 3 && parameters[0]=="PASS" ){
			Add("white",parameters[1], uint64(ToNumber(parameters[2])) , false)
			return "WHITED:" + parameters[1] + ":" + parameters[2]
		}
		return "NO_ACTION("+request+")"
	})
}



func CheckIsClosed(){
	changed, err := IsProgramChanged()
	if err != nil {
		fmt.Printf("[X] Program kontrolü: %v\n", err)
	} else if changed {
		fmt.Printf("[!] XDP programı değişti, map'ler yeniden initialize ediliyor\n")
		if err := ReInitMaps(); err != nil {
			fmt.Printf("[X] ReInitMaps: %v\n", err)
		} else {
			fmt.Printf("[+] XDP programı ve .bss yeniden yüklendi\n")
		}
	}
}