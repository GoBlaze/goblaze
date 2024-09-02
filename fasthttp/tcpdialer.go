package fasthttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	zenq "github.com/GoBlaze/goblaze/chan"
	"github.com/alphadose/haxmap"
)

func Dial(addr string) (net.Conn, error) {
	return defaultDialer.Dial(addr)
}

func DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	return defaultDialer.DialTimeout(addr, timeout)
}

func DialDualStack(addr string) (net.Conn, error) {
	return defaultDialer.DialDualStack(addr)
}

func DialDualStackTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	return defaultDialer.DialDualStackTimeout(addr, timeout)
}

var defaultDialer = NewTCPDialer[string, int](1000, nil)

type Resolver interface {
	LookupIPAddr(context.Context, string) (names []net.IPAddr, err error)
}

func NewTCPDialer[T hashable, V any](concurrency int, resolver Resolver) *TCPDialer[T, V] {
	return &TCPDialer[T, V]{
		Resolver:         resolver,
		Concurrency:      concurrency,
		DNSCacheDuration: DefaultDNSCacheDuration,
		ma:               haxmap.New[T, any](),
	}
}

type TCPDialer[T hashable, V any] struct {
	Resolver             Resolver
	once                 sync.Once
	LocalAddr            *net.TCPAddr
	concurrencyCh        chan struct{}
	Concurrency          int
	DNSCacheDuration     time.Duration
	DisableDNSResolution bool
	ma                   *haxmap.Map[T, any]
}

func (d *TCPDialer[T, V]) Dial(addr T) (net.Conn, error) {
	return d.dial(addr, false, DefaultDialTimeout)
}

func (d *TCPDialer[T, V]) DialTimeout(addr T, timeout time.Duration) (net.Conn, error) {
	return d.dial(addr, false, timeout)
}

func (d *TCPDialer[T, V]) DialDualStack(addr T) (net.Conn, error) {
	return d.dial(addr, true, DefaultDialTimeout)
}

func (d *TCPDialer[T, V]) DialDualStackTimeout(addr T, timeout time.Duration) (net.Conn, error) {
	return d.dial(addr, true, timeout)
}

func (d *TCPDialer[T, V]) dial(addr T, dualStack bool, timeout time.Duration) (net.Conn, error) {
	d.once.Do(func() {
		if d.Concurrency > 0 {
			d.concurrencyCh = make(chan struct{}, d.Concurrency)
		}

		if d.DNSCacheDuration == 0 {
			d.DNSCacheDuration = DefaultDNSCacheDuration
		}

		if !d.DisableDNSResolution {
			go d.tcpAddrsClean()
		}
	})
	deadline := time.Now().Add(timeout)
	network := "tcp4"
	if dualStack {
		network = "tcp"
	}
	if d.DisableDNSResolution {
		return d.tryDial(network, addr, deadline, d.concurrencyCh)
	}
	addrs, idx, err := d.getTCPAddrs(addr, dualStack, deadline)
	if err != nil {
		return nil, err
	}
	var conn net.Conn
	n := uint32(len(addrs))
	for n > 0 {
		addr := any(addrs[idx%n].String()).(T)
		conn, err = d.tryDial(network, addr, deadline, d.concurrencyCh)
		if err == nil {
			return conn, nil
		}
		if errors.Is(err, ErrDialTimeout) {
			return nil, err
		}
		idx++
		n--
	}
	return nil, err
}

func (d *TCPDialer[T, V]) tryDial(network string, addr T, deadline time.Time, concurrencyCh chan struct{}) (net.Conn, error) {
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, wrapDialWithUpstream(ErrDialTimeout, fmt.Sprint(addr))
	}

	if concurrencyCh != nil {
		select {
		case concurrencyCh <- struct{}{}:
		default:
			tc := AcquireTimer(timeout)
			isTimeout := false

			concurrencyCh <- struct{}{}

			if data := zenq.Select(tc.C); data != nil {
				switch data.(type) {
				case time.Time:
					isTimeout = true
				}
			}
			ReleaseTimer(tc)
			if isTimeout {
				return nil, wrapDialWithUpstream(ErrDialTimeout, fmt.Sprint(addr))
			}
		}
		func() { <-concurrencyCh }()
	}

	dialer := net.Dialer{}
	if d.LocalAddr != nil {
		dialer.LocalAddr = d.LocalAddr
	}

	ctx, cancelCtx := context.WithDeadline(context.Background(), deadline)
	defer cancelCtx()
	conn, err := dialer.DialContext(ctx, network, fmt.Sprint(addr))
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, wrapDialWithUpstream(ErrDialTimeout, fmt.Sprint(addr))
		}
		return nil, wrapDialWithUpstream(err, fmt.Sprint(addr))
	}
	return conn, nil
}

var ErrDialTimeout = errors.New("dialing to the given TCP address timed out")

type ErrDialWithUpstream struct {
	wrapErr  error
	Upstream string
}

func (e *ErrDialWithUpstream) Error() string {
	return fmt.Sprintf("error when dialing %s: %s", e.Upstream, e.wrapErr.Error())
}

func (e *ErrDialWithUpstream) Unwrap() error {
	return e.wrapErr
}

func wrapDialWithUpstream(err error, upstream string) error {
	return &ErrDialWithUpstream{
		Upstream: upstream,
		wrapErr:  err,
	}
}

const DefaultDialTimeout = 3 * time.Second

type tcpAddrEntry struct {
	resolveTime time.Time
	addrs       []net.TCPAddr
	addrsIdx    uint32
	pending     int32
}

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
	Signed | Unsigned
}

type Float interface {
	~float32 | ~float64
}

type Complex interface {
	~complex64 | ~complex128
}

type Ordered interface {
	Integer | Float | ~string
}

type hashable interface {
	Integer | Float | Complex | ~string | uintptr | ~unsafe.Pointer
}

// DefaultDNSCacheDuration is the duration for caching resolved TCP addresses by Dial* functions.
const DefaultDNSCacheDuration = time.Minute

func (d *TCPDialer[T, V]) tcpAddrsClean() {
	expireDuration := 2 * d.DNSCacheDuration
	for {
		time.Sleep(time.Second)
		t := time.Now()
		d.ma.ForEach(func(k T, v any) bool {
			if e, ok := v.(*tcpAddrEntry); ok && t.Sub(e.resolveTime) > expireDuration {
				d.ma.Del(k)
				return true
			}
			return false
		})
	}
}

func (d *TCPDialer[T, V]) getTCPAddrs(addr T, dualStack bool, deadline time.Time) ([]net.TCPAddr, uint32, error) {
	item, exist := d.ma.Get(addr)
	e, ok := item.(*tcpAddrEntry)
	if exist && ok && e != nil && time.Since(e.resolveTime) > d.DNSCacheDuration {
		// Only let one goroutine re-resolve at a time.
		if atomic.SwapInt32(&e.pending, 1) == 0 {
			e = nil
		}
	}

	if e == nil {
		addrs, err := resolveTCPAddrs(fmt.Sprint(addr), dualStack, d.Resolver, deadline)
		if err != nil {
			item, exist := d.ma.Get(addr)
			e, ok = item.(*tcpAddrEntry)
			if exist && ok && e != nil {
				// Set pending to 0 so another goroutine can retry.
				atomic.StoreInt32(&e.pending, 0)
			}
			return nil, 0, err
		}
		e = &tcpAddrEntry{
			addrs:       addrs,
			resolveTime: time.Now(),
		}
		d.ma.Set(addr, e)
	}

	idx := atomic.AddUint32(&e.addrsIdx, 1)
	return e.addrs, idx, nil
}

func resolveTCPAddrs(addr string, dualStack bool, resolver Resolver, deadline time.Time) ([]net.TCPAddr, error) {
	host, portS, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}

	if resolver == nil {
		resolver = net.DefaultResolver
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	ipaddrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	n := len(ipaddrs)
	addrs := make([]net.TCPAddr, 0, n)
	for i := 0; i < n; i++ {
		ip := ipaddrs[i]
		if !dualStack && ip.IP.To4() == nil {
			continue
		}
		addrs = append(addrs, net.TCPAddr{
			IP:   ip.IP,
			Port: port,
			Zone: ip.Zone,
		})
	}
	if len(addrs) == 0 {
		return nil, errNoDNSEntries
	}
	return addrs, nil
}

var errNoDNSEntries = errors.New("couldn't find DNS entries for the given domain. Try using DialDualStack")
