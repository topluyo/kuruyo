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
  RATE_SIZE_OPEN     = 12
  RATE_SIZE_WINDOW   = 11
  RATE_SIZE_REQUEST  = 8
  RATE_SIZE_BLOCKED  = 1
)



const(
  RATE_OFFSET_OPEN     = 0
  RATE_OFFSET_WINDOW   = RATE_OFFSET_OPEN    + RATE_SIZE_OPEN
  RATE_OFFSET_REQUEST  = RATE_OFFSET_WINDOW  + RATE_SIZE_WINDOW
  RATE_OFFSET_BLOCKED  = RATE_OFFSET_REQUEST + RATE_SIZE_REQUEST

  MASK_OPEN    = (1 << RATE_SIZE_OPEN) - 1
  HALF_OPEN      = 1 << (RATE_SIZE_OPEN - 1)
  MASK_WINDOW  = (1 << RATE_SIZE_WINDOW) - 1
  MASK_REQUEST = (1 << RATE_SIZE_REQUEST) - 1
  MASK_BLOCKED = (1 << RATE_SIZE_BLOCKED) - 1
)


func CoolDownPassed(now, open uint16, mask, half uint16) bool {
  d := (now - open) & mask
  return d != 0 && d < half
}

//@ Limit
type Limit struct{
  Info      string
	Request   uint16
  Period    uint32
  Wait      uint32
  
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

    write("├─── RateLimit : " + key)

    RateParams := strings.Split(rate, " ")
    if len(RateParams) < 3 {
      write("[X] Rate parameters error on \"" + rate + "\"")
      continue
    }

    request := ToNumber(RateParams[0])
    second  := ToNumber(RateParams[1])
    wait    := ToNumber(RateParams[2])
    
    
    limit   :=  &Limit{
      Info     : rate,
      Request  : uint16(request),
      Period   : uint32(second),
      Wait     : uint32(wait),
      Connects : make([]uint32, 1<<RATE_BITS),
    }
    DefineWindow(limit)

    newLimiter[key] = limit

    write("   └──", ToString(request)+"r", ToString(second)+"s", ToString(wait)+"w")
  
  
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




func GetRateConnect(index uint64, limit *Limit) (open uint16, window uint16, request uint16, blocked uint16, currentWindow uint16, currentOpen uint16) {
	value := atomic.LoadUint32(&limit.Connects[index])
	open = uint16(value & MASK_OPEN)
  window = uint16((value >> RATE_OFFSET_WINDOW) & MASK_WINDOW)
  request = uint16((value >> RATE_OFFSET_REQUEST) & MASK_REQUEST)
  blocked = uint16((value >> RATE_OFFSET_BLOCKED) & MASK_BLOCKED)

	currentWindow = uint16(limit.Window.Load() & MASK_WINDOW)
	currentOpen   = uint16(limit.Open.Load() & MASK_OPEN)
	return
}


func SetRateConnect(index uint64, limit *Limit, open, window, request, blocked uint16) {
  value := uint32(open&MASK_OPEN) |
    (uint32(window&MASK_WINDOW) << RATE_OFFSET_WINDOW) |
    (uint32(request&MASK_REQUEST) << RATE_OFFSET_REQUEST) |
    (uint32(blocked&MASK_BLOCKED) << RATE_OFFSET_BLOCKED)
	atomic.StoreUint32(&limit.Connects[index], value)
}

func PackRateConnect(open, window, request, blocked uint16) uint32 {
	return uint32(open&MASK_OPEN) |
    (uint32(window&MASK_WINDOW) << RATE_OFFSET_WINDOW) |
    (uint32(request&MASK_REQUEST) << RATE_OFFSET_REQUEST) |
    (uint32(blocked&MASK_BLOCKED) << RATE_OFFSET_BLOCKED)
}

func UpdateRateConnect(index uint64, limit *Limit, currentWindow uint16, currentOpen uint16, maxRequest uint16) bool {

	for {
		old := atomic.LoadUint32(&limit.Connects[index])


    open    := uint16(old & MASK_OPEN)
    window  := uint16((old >> RATE_OFFSET_WINDOW) & MASK_WINDOW)
    request := uint16((old >> RATE_OFFSET_REQUEST) & MASK_REQUEST)
    blocked := uint16((old >> RATE_OFFSET_BLOCKED) & MASK_BLOCKED)


		if blocked == 1 {
			if !CoolDownPassed(currentOpen, open, MASK_OPEN, HALF_OPEN) {
				return false
			}
			blocked = 0
			request = 0
			window = currentWindow
		}


		if window != currentWindow {
			window = currentWindow
			request = 0
		}


		if request >= maxRequest {
			blocked   = 1
      open = (currentOpen + 1) & MASK_OPEN
			newValue := PackRateConnect(open,window,request,blocked)
			if atomic.CompareAndSwapUint32(&limit.Connects[index],old,newValue) {
				return false
			}

			continue
		}

		request++
		newValue := PackRateConnect(open,window,request,blocked)
		if atomic.CompareAndSwapUint32(&limit.Connects[index],old,newValue) {
			return true
		}
	}
}

func CheckRateLimiter(ip string, limits []*Limit) bool {

	ipset := Rate_IPSET(ip)

	for _, limit := range limits {
		currentWindow := uint16(limit.Window.Load() & MASK_WINDOW)
		currentOpen   := uint16(limit.Open.Load()   & MASK_OPEN)
		if !UpdateRateConnect(ipset,limit,currentWindow,currentOpen,limit.Request) {
			return false
		}
	}
	return true
}