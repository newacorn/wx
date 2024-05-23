package xff

import (
	"net"
	"slices"
	"strconv"
	"strings"

	routing "fasthttp-routing"
	"github.com/rs/zerolog/log"
	"helpers/unsafefn"
	"helpers/utilconvert"
	"helpers/utilnet"
)

type Config struct {
	TrustProxies     []string
	iPRanges         []net.IPNet
	Skip             func(c *routing.Ctx) bool
	TrustedHeaderSet int
}

var DefCfg = Config{
	TrustedHeaderSet: routing.HEADER_X_FORWARDED_FOR | routing.HEADER_X_FORWARDED_HOST | routing.HEADER_X_FORWARDED_PORT | routing.HEADER_X_FORWARDED_PROTO,
}

const (
	FROMUNKNOWN   = 0
	FROMUNTRUSTED = 1
	FROMTRUST     = 2
)

type xffInfo struct {
	cfg   *Config
	trust int
}

func (x *xffInfo) isFromTrustedProxy(c *routing.Ctx) bool {
	if x.trust != FROMUNKNOWN {
		return x.trust == FROMTRUST
	}
	if len(x.cfg.TrustProxies) > 0 && checkIp(c, x.cfg) {
		x.trust = FROMTRUST
		return true
	}
	x.trust = FROMUNTRUSTED
	return false
}
func (x *xffInfo) Host(ctx *routing.Ctx) {
	if !x.isFromTrustedProxy(ctx) {
		return
	}

	var forwardHost []byte
	if routing.HEADER_X_FORWARDED_HOST&x.cfg.TrustedHeaderSet != 0 {
		forwardHost = ctx.Request.Header.Peek(routing.HeaderXForwardedHost)
	}
	if len(forwardHost) == 0 && x.cfg.TrustedHeaderSet&routing.HEADER_FORWARDED != 0 {
		ff := ctx.Request.Header.Peek(routing.HeaderForwarded)
	outer:
		for i := range ff {
			if ff[i] == 'h' || ff[i] == 'H' {
				if len(ff)-i < 5 {
					break
				}
				part := ff[i : i+5]
				utilconvert.ToLower(part)
				if unsafefn.BtoS(part) == "host=" {
					after := ff[i+5:]
					var j int
					for j = range after {
						if after[j] == ',' || after[j] == ';' {
							forwardHost = after[:j]
							break outer
						}
					}
					forwardHost = after
					break outer
				}
			}
		}
	}
	if len(forwardHost) != 0 {
		ip, port, err := utilnet.SplitIpAndPort(unsafefn.BtoS(forwardHost))
		if err != nil {
			return
		}
		if len(port) != 0 {
			i, err := strconv.Atoi(port)
			if err == nil {
				ctx.RPort = -i
			}
		}
		ctx.RHost = append(ctx.RHost, ip...)
	}

}

func (x *xffInfo) Port(ctx *routing.Ctx) {
	if !x.isFromTrustedProxy(ctx) {
		return
	}
	x.Host(ctx)
	var portByte []byte
	if routing.HEADER_X_FORWARDED_PORT&x.cfg.TrustedHeaderSet != 0 {
		portByte = ctx.Request.Header.Peek(routing.HeaderXForwardedPort)
	}
	if len(portByte) == 0 {
		return
	}
	i, err := strconv.Atoi(unsafefn.BtoS(portByte))
	if err == nil {
		ctx.RPort = i
	}
}

func (x *xffInfo) IP(ctx *routing.Ctx) {
	if !x.isFromTrustedProxy(ctx) {
		return
	}
	var xff []byte
	if routing.HEADER_X_FORWARDED_FOR&x.cfg.TrustedHeaderSet != 0 {
		xff = ctx.Request.Header.Peek(routing.HeaderXForwardedFor)
	}
	var xffs []string
	if len(xff) != 0 {
		xffs = strings.Split(unsafefn.BtoS(xff), ",")
	}
	if len(xff) == 0 && x.cfg.TrustedHeaderSet&routing.HEADER_FORWARDED != 0 {
		ff := ctx.Request.Header.Peek(routing.HeaderForwarded)
		if len(ff) == 0 {
			return
		}
	outer:
		for i := range ff {
			if ff[i] == 'f' || ff[i] == 'F' {
				if len(ff)-i < 4 {
					break
				}
				part := ff[i : i+4]
				utilconvert.ToLower(part)
				if unsafefn.BtoS(part) == "for=" {
					after := ff[i+4:]
					var j int
					for j = range after {
						if after[j] == ',' || after[j] == ';' {
							xffs = append(xffs, unsafefn.BtoS(after[:j]))
							i = i + j
							continue outer
						}
					}
					xffs = append(xffs, unsafefn.BtoS(after))
					i = i + j
				}
			}

		}
	}
	if len(xffs) == 0 {
		return
	}
	realIp := normalizeAndFilterClientIps(xffs, x.cfg.iPRanges)
	if len(realIp) == 0 {
		return
	}
	ctx.RIP = append(ctx.RIP, realIp...)
}

func (x *xffInfo) Secure(ctx *routing.Ctx) {
	if !x.isFromTrustedProxy(ctx) {
		return
	}
	var xfp []byte
	if routing.HEADER_X_FORWARDED_PROTO&x.cfg.TrustedHeaderSet != 0 {
		xfp = ctx.Request.Header.Peek(routing.HeaderXForwardedProto)
	}
	if len(xfp) == 0 && x.cfg.TrustedHeaderSet&routing.HEADER_FORWARDED != 0 {

		ff := ctx.Request.Header.Peek(routing.HeaderForwarded)
	outer:
		for i := range ff {
			if ff[i] == 'p' || ff[i] == 'P' {
				if len(ff)-i < 6 {
					break
				}
				part := ff[i : i+6]
				utilconvert.ToLower(part)
				if unsafefn.BtoS(part) == "proto=" {
					after := ff[i+6:]
					var j int
					for j = range after {
						if after[j] == ',' || after[j] == ';' {
							xfp = after[:j]
							break outer
						}
					}
					xfp = after
					break outer
				}
			}
		}
	}
	if len(xfp) != 0 {
		if slices.Contains([]string{"https", "on", "ssl", "1"}, unsafefn.BtoS(xfp)) {
			ctx.RSecure = routing.PROTOSECURE
			return
		}
		ctx.RSecure = routing.PROTOUNSECURE
		return
	}
}

func New(cfgs ...Config) routing.Handler {
	// var ipRanges []net.IPNet
	var cfg *Config
	if len(cfgs) == 0 {
		cfg = &DefCfg
	} else {
		cfg = &cfgs[0]
	}
	if len(cfg.TrustProxies) > 0 {
		ipRanges, err := utilnet.AddressesAndRangesToIPNets(cfg.TrustProxies...)
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("parse ip ranges in xff middleware")
		}
		cfg.iPRanges = ipRanges
	}
	return func(c *routing.Ctx) error {
		if cfg.Skip != nil && cfg.Skip(c) {
			return c.Next()
		}
		c.XFFInfo = &xffInfo{cfg: cfg}
		return c.Next()
	}
}
func checkIp(ctx *routing.Ctx, cfg *Config) bool {
	rAddr := ctx.RemoteAddr()
	var rIp net.IP
	remoteIp, ok := rAddr.(*net.TCPAddr)
	if ok {
		rIp = remoteIp.IP
	} else {
		remoteIp2, ok := rAddr.(*net.UDPAddr)
		if ok {
			rIp = remoteIp2.IP
		}
	}
	return checkIp2(rIp, cfg.iPRanges)
}
func checkIp2(ip net.IP, ranges []net.IPNet) bool {
	for _, ipRange := range ranges {
		if ipRange.Contains(ip) {
			return true
		}
	}
	return false
}

func normalizeAndFilterClientIps(ips []string, ipRange []net.IPNet) (realIp string) {
	l := len(ips)
	if l == 0 {
		return
	}
outer:
	for i := l - 1; i >= 0; i-- {
		ip2, _, err := utilnet.SplitIpAndPort(ips[i])
		if err != nil {
			continue
		}
		ipB := net.ParseIP(ip2)
		if ipB == nil {
			continue
		}
		for i := range ipRange {
			if ipRange[i].Contains(ipB) {
				continue outer
			}
		}
		return ipB.String()
	}
	return
}
