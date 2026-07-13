package main

import(
  "sync/atomic"
  "context"
  "strings"
  "time"
)


var RATE_BITS int = 25

//- BLOCKS : uint32

const (
  RATE_SIZE_REQUEST uint32 = 8
  RATE_SIZE_OPEN    uint32 = 12
  RATE_SIZE_WINDOW  uint32 = 11
  RATE_SIZE_BLOCKED uint32 = 1
)

const(
  RATE_OFFSET_REQUEST uint32 = 0
  RATE_OFFSET_OPEN    uint32 = RATE_OFFSET_REQUEST + RATE_SIZE_REQUEST
  RATE_OFFSET_WINDOW  uint32 = RATE_OFFSET_OPEN    + RATE_SIZE_OPEN
  RATE_OFFSET_BLOCKED uint32 = RATE_OFFSET_WINDOW  + RATE_SIZE_WINDOW
)


const(
  VALUE_MASK_REQUEST uint32 = (1 << RATE_SIZE_REQUEST) - 1
  VALUE_MASK_OPEN    uint32 = (1 << RATE_SIZE_OPEN) - 1
  VALUE_MASK_WINDOW  uint32 = (1 << RATE_SIZE_WINDOW) - 1
  VALUE_MASK_BLOCKED uint32 = (1 << RATE_SIZE_BLOCKED) - 1
)

const (
	AREA_MASK_REQUEST  uint32 = VALUE_MASK_REQUEST << RATE_OFFSET_REQUEST
	AREA_MASK_OPEN     uint32 = VALUE_MASK_OPEN    << RATE_OFFSET_OPEN
	AREA_MASK_WINDOW   uint32 = VALUE_MASK_WINDOW  << RATE_OFFSET_WINDOW
	AREA_MASK_BLOCKED  uint32 = VALUE_MASK_BLOCKED << RATE_OFFSET_BLOCKED


	NOT_AREA_MASK_REQUEST  uint32 = ^AREA_MASK_REQUEST
	NOT_AREA_MASK_OPEN     uint32 = ^AREA_MASK_OPEN
	NOT_AREA_MASK_WINDOW   uint32 = ^AREA_MASK_WINDOW
	NOT_AREA_MASK_BLOCKED  uint32 = ^AREA_MASK_BLOCKED
)
const(
  HALF_OPEN uint32 = 1 << (RATE_SIZE_OPEN - 1)
)


func CoolDownPassed(now, open uint32, mask, half uint32) bool {
  d := (now - open) & mask
  return d != 0 && d < half
}

//@ Limit
type Limit struct{
  Info      string
	Request   uint32
  Period    uint32
  Wait      uint32
	Status    int
  
  Connects  []uint32

  Window          atomic.Uint32
  WindowCancel    context.CancelFunc
  Open            atomic.Uint32
  OpenCancel      context.CancelFunc
}

// Eşzamanlılık için
type RateLimiter struct {
	limits   map[string]*Limit
}
var Limiter atomic.Pointer[RateLimiter]

//@ DefineLimits
func DefineRateLimit(){
  table("RateLimit")

	existingLimiter := Limiter.Load()
	if existingLimiter == nil {
		write("└── DefineRateLimit()")
	} else {
		write("└── ReloadRateLimit()")
    for _, limit := range existingLimiter.limits {
      if limit.WindowCancel != nil {
        limit.WindowCancel()
      }
      if limit.OpenCancel != nil {
        limit.OpenCancel()
      }
    }
	}

	newLimiter := make(map[string]*Limit)
  if(server.RateSIZE==0){
    server.RateSIZE = 20
  }
  RATE_BITS = server.RateSIZE

	for key, level := range server.Levels {
		rate := level.Rates
		if(rate!=""){

			write("├─── RateLimit : " + key)

			RateParams := strings.Split(rate, " ")
			if len(RateParams) < 3 {
				write("[X] Rate parameters error on \"" + rate + "\"")
				continue
			}

			request := ToNumber(RateParams[0])
			second  := ToNumber(RateParams[1])
			wait    := ToNumber(RateParams[2])
			status  := 0
			if(len(RateParams)==4){
				status = ToNumber(RateParams[3])
			}
			
			
			limit   :=  &Limit{
				Info     : rate,
				Request  : uint32(request),
				Period   : uint32(second),
				Wait     : uint32(wait),
				Status   : int(status),
				Connects : make([]uint32, 1<<RATE_BITS,1<<RATE_BITS),
			}
			DefineWindow(limit)

			newLimiter[key] = limit

			write("   └──", ToString(request)+"r", ToString(second)+"s", ToString(wait)+"w")
		
		}
  
	}

	for name := range server.Routes {
		r := server.Routes[name]
    r.Limits = make([]*Limit, 0)
    r.UseLimit = false
		for _, level := range r.Levels {
			if limit, ok := newLimiter[level]; ok {
        write("├─── Install RateLimit(" + name + " " + level + " " + limit.Info + ")")
        r.Limits = append(r.Limits, limit)
				r.UseLimit = true
			}
		}
	}

	Limiter.Store(&RateLimiter{ limits: newLimiter})

	//RateLimiterChecker()

}

func DefineWindow(l *Limit) {
	WindowCTX, WindowCANCEL := context.WithCancel(context.Background())
	l.WindowCancel = WindowCANCEL
	windowTicker := time.NewTicker( time.Duration(l.Period) * time.Second)
	go func() {
		defer windowTicker.Stop()
    window := uint32(time.Now().Unix() / int64(l.Period))
    l.Window.Store(window)
		for {
			select {
			case now := <-windowTicker.C:
        window := uint32(now.Unix() / int64(l.Period))
				l.Window.Store(window)
			case <-WindowCTX.Done():
				return
			}
		}
	}()


  OpenCTX, OpenCANCEL := context.WithCancel(context.Background())
	l.OpenCancel = OpenCANCEL
	openTicker := time.NewTicker( time.Duration(l.Wait) * time.Second)
	go func() {
		defer openTicker.Stop()
    Open := uint32(time.Now().Unix() / int64(l.Wait))
    l.Open.Store(Open)
		for {
			select {
			case now := <-openTicker.C:
        Open := uint32(now.Unix() / int64(l.Wait))
				l.Open.Store(Open)
			case <-OpenCTX.Done():
				return
			}
		}
	}()
}



func Rate_IPSET(ip string) uint64 {
	var h uint64 = 2166136261 // FNV offset basis
	for i := 0; i < len(ip); i++ {
		h ^= uint64(ip[i])
		h *= 16777619
	}
	return h & ((1 << RATE_BITS) - 1)
}




func UpdateRateConnect(index uint64, limit *Limit, currentWindow uint32, currentOpen uint32, maxRequest uint32, status int) bool {

	var old uint32

	if(status==0){
		old = atomic.AddUint32(&limit.Connects[index], 1)
	}else{
		old = atomic.LoadUint32(&limit.Connects[index])
	}


	request   := old & VALUE_MASK_REQUEST 
	open      := (old >> RATE_OFFSET_OPEN) & VALUE_MASK_OPEN 
	window    := (old >> RATE_OFFSET_WINDOW) & VALUE_MASK_WINDOW 
	blocked   := (old >> RATE_OFFSET_BLOCKED) & VALUE_MASK_BLOCKED 

	//write(request,open,window,blocked,currentWindow,currentOpen,maxRequest)

	if blocked == 1 {
		if !CoolDownPassed(currentOpen, open, VALUE_MASK_OPEN, HALF_OPEN) {
			return false
		}
		// request=1, open=0, window=0, blocked=0
		atomic.StoreUint32(&limit.Connects[index], 1)
		return true
	}


	if window != currentWindow {
		// request=1
		atomic.AndUint32(&limit.Connects[index], NOT_AREA_MASK_REQUEST)
		atomic.OrUint32(&limit.Connects[index], 1)
		// window=currentWindow
		atomic.AndUint32(&limit.Connects[index], NOT_AREA_MASK_WINDOW)
		atomic.OrUint32(&limit.Connects[index], AREA_MASK_WINDOW & (currentWindow << RATE_OFFSET_WINDOW ))
		return true
	}


	if request > maxRequest {
		// blocked=1
		atomic.OrUint32(&limit.Connects[index], AREA_MASK_BLOCKED)
		// open=currentOpen+1
		open = ((currentOpen + 1) & VALUE_MASK_OPEN) << RATE_OFFSET_OPEN

		atomic.AndUint32(&limit.Connects[index], NOT_AREA_MASK_OPEN)
		atomic.OrUint32(&limit.Connects[index], open )
		
		return false
	}

	return true
}

func CheckRateLimiter(ip string, limits []*Limit, status int) bool {

	ipset := Rate_IPSET(ip)

	for _, limit := range limits {
		currentWindow := limit.Window.Load() & VALUE_MASK_WINDOW
		currentOpen   := limit.Open.Load()   & VALUE_MASK_OPEN
		if !UpdateRateConnect(ipset,limit,currentWindow,currentOpen, limit.Request,status^limit.Status ) {
			return false
		}
	}
	return true
}
