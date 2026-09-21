
//!!! NONE USING

package main

import (
	"log"
	"github.com/cilium/ebpf"
	"fmt"
	
)







var statsMap *ebpf.Map
func InitMinuteSetter(){
	prog, err := ebpf.LoadPinnedProgram("/sys/fs/bpf/bozkurt_program",nil,)
  if err != nil {
		fmt.Println("[X] LoadPinnedProgram:", err)
		return
  }
  defer prog.Close()

	info, err := prog.Info()
	if err != nil {
		log.Fatalf("Program bilgisi alınamadı: %v", err)
	}

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
    if ( err==nil ){
      log.Println(mInfo.Name)
    }
		if err == nil && mInfo.Name == ".bss" {
			statsMap = m
			break
		}
		m.Close()
	}

	if statsMap == nil {
		log.Fatalf(".bss haritası bulunamadı")
	}
}


func main(){
  InitMinuteSetter()
}