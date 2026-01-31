package main

import (
  "runtime"
  "fmt"
  "log"
  "time"
  "sync/atomic"

)



type Block[T any] struct {
	Size  int64
	List  []T
}


func NewBlock[T any](size int) *Block[T] {
	return &Block[T]{
		Size: 1 << size - 1,
		List: make([]T,1 << size),
	}
}

func (b *Block[T]) Get(key int64) T {
  hash := key & b.Size
  log.Println(key, b.Size,hash)
  log.Println(hash)
  val := b.List[hash]
  return val
}

func (b *Block[T]) Find(key int64) T {
  hash := key & b.Size
  return b.List[hash]
  var zero T
  b.List[hash] = zero
  return zero
}


func (b *Block[T]) Set(key int64,val T) {
  hash := key & b.Size  
  b.List[hash] = val
}



type Rate struct{
  Count  int
  Window int64
}


type Window struct{
  Value int64
}

func (w *Window) Get() int64{
  return atomic.LoadInt64(&w.Value)
}


func NewWindow(period int64) *Window {
    w := &Window{}

    ticker := time.NewTicker(1 * time.Second)

    go func() {
        defer ticker.Stop()
        for range ticker.C {
            atomic.StoreInt64(&w.Value, time.Now().Unix()/period)
        }
    }()

    return w
}


var black *Block[Rate]
func main(){

  
  black = NewBlock[Rate](22)
  log.Println(black.Size)
  

  var rate Rate = black.Get(int64(124012398098))

  log.Println(rate)


  log.Printf("------------------------")
  // Read memory stats
  var m runtime.MemStats
  runtime.ReadMemStats(&m)
  fmt.Printf("Alloc = %v MB\n", m.Alloc/1048576)       // currently allocated heap memory
  fmt.Printf("TotalAlloc = %v MB\n", m.TotalAlloc/1048576) // total allocated heap memory
  fmt.Printf("Sys = %v MB\n", m.Sys/1048576)           // total memory obtained from OS
  fmt.Printf("NumGC = %v\n", m.NumGC)    
}



func benc(){

}