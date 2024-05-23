package xff

import (
	"net"
	"testing"

	routing "fasthttp-routing"
	"github.com/newacorn/fasthttp"
	"github.com/stretchr/testify/assert"
	"helpers/unsafefn"
)

var router *routing.Router

func init() {
}

type out struct {
	IP    string
	Port  int
	Host  string
	Proto string
}
type input struct {
	name            string
	remoteIp        string
	remotePort      int
	HeaderHost      string
	HeaderXFF       string
	HeaderXFH       string
	HeaderXFP       string
	HeaderXFPRO     string
	TrustProxies    []string
	TrustHeadersSet int
	FF              string
	want            out
}

func TestHost(t *testing.T) {

	for _, in := range []input{
		{name: "without xff", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			TrustHeadersSet: DefCfg.TrustedHeaderSet, want: out{IP: "127.0.0.1", Port: 3344, Host: "127.0.0.1", Proto: routing.HTTP}},
		{name: "with xff", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			HeaderXFF:    "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 3344, Host: "127.0.0.1", Proto: routing.HTTP}},
		{name: "with xff with port", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			HeaderXFF:    "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 3344, Host: "127.0.0.1", Proto: routing.HTTP}},
		{name: "with xff with xfh", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			HeaderXFH:    "192.168.33.44:6743",
			HeaderXFF:    "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 6743, Host: "192.168.33.44", Proto: routing.HTTP}},
		{name: "with xff+port with xfh  with xfp with proto", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			HeaderXFF: "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			HeaderXFP: "8877", HeaderXFH: "baidu.com:3333", HeaderXFPRO: "on",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 8877, Host: "baidu.com", Proto: routing.HTTPS}},
		{name: "with xff+port with xfh  with xfp with proto", remoteIp: "127.0.0.1", remotePort: 3099, HeaderHost: "127.0.0.1:3344",
			HeaderXFF: "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			HeaderXFP: "8877", HeaderXFH: "baidu.com:3333", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 8877, Host: "baidu.com", Proto: routing.HTTPS}},
		{name: "with xff+port with xfh  with xfp with proto host without host", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "127.0.0.1:3344",
			HeaderXFF:  "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			HeaderXFH:  "baidu.com", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 3344, Host: "baidu.com", Proto: routing.HTTPS}},
		{name: "port form scheme", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "127.0.0.1",
			HeaderXFF:  "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,198.51.100.178:8888",
			HeaderXFH:  "baidu.com", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "198.51.100.178", Port: 443, Host: "baidu.com", Proto: routing.HTTPS}},
		{name: "ip6 addrs", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "127.0.0.1",
			HeaderXFF:  "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,[2001:db8:1f70::999:de8:7648:6e8]:8888",
			HeaderXFH:  "baidu.com", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e8", Port: 443, Host: "baidu.com", Proto: routing.HTTPS}},
		{name: "ip6 addrs omit []", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "127.0.0.1",
			HeaderXFF:  "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,[2001:db8:1f70::999:de8:7648:6e8]:8888",
			HeaderXFH:  "2001:db8:1f70::999:de8:7648:6e8:100", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e8", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8", Proto: routing.HTTPS}},
		{name: "ip6 addrs with [] without port", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "127.0.0.1",
			HeaderXFF:  "203.0.113.195,2001:db8:85a3:8d3:1319:8a2e:370:7348,[2001:db8:1f70::999:de8:7648:6e8]",
			HeaderXFH:  "[2001:db8:1f70::999:de8:7648:6e8]", HeaderXFPRO: "https",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet,
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e8", Port: 443, Host: "2001:db8:1f70::999:de8:7648:6e8", Proto: routing.HTTPS}},
		{name: "hostHeader with ip6", remoteIp: "2001:db8:1f70:0:999:de8:7648:6e83", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]:7777",
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e83", Port: 7777, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},
		{name: "hostHeader with ip6 without port", remoteIp: "2001:db8:1f70:0:999:de8:7648:6e83", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]",
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e83", Port: 80, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},
		{name: "hostHeader with ip6 without port", remoteIp: "2001:db8:1f70:0:999:de8:7648:6e83", remotePort: 3099,
			HeaderHost: "2001:db8:1f70::999:de8:7648:6e8:100",
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e83", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},
		{name: "hostHeader with ip6 without port", remoteIp: "2001:db8:1f70:0:999:de8:7648:6e83", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]:100", FF: "For=[2001:db8:cafe::17]:4711",
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e83", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},
		{name: "hostHeader with ip6 without port", remoteIp: "2001:db8:1f70:0:999:de8:7648:6e83", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]:100", FF: "For=[2001:db8:cafe::17]:4711",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want: out{IP: "2001:db8:1f70:0:999:de8:7648:6e83", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},
		{name: "hostHeader with ip6 without port", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]:100", FF: "For=[2001:db8:cafe::17]:4711",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want: out{IP: "2001:db8:cafe::17", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8",
				Proto: routing.HTTP}},

		{name: "xff with proto", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost: "[2001:db8:1f70::999:de8:7648:6e8]:100", FF: "fOr=[2001:db8:cafe::17]:4711;proTo=https;Host=baidu.com:7777",
			TrustProxies: []string{"127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want: out{IP: "2001:db8:cafe::17", Port: 7777, Host: "baidu.com",
				Proto: routing.HTTPS}},
		{name: "xff with multi xff", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost:   "[2001:db8:1f70::999:de8:7648:6e8]:100",
			FF:           "for=192.168.33.1:834, fOr=[2001:db8:cafe::17]:4711;proTo=https;Host=baidu.com:7777",
			TrustProxies: []string{"2001:db8:cafe::17", "127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want: out{IP: "192.168.33.1", Port: 7777, Host: "baidu.com",
				Proto: routing.HTTPS}},
		{name: "xff with case insensitive", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost:   "[2001:db8:1f70::999:de8:7648:6e8]:100",
			FF:           "for=192.168.33.1:834, fOr=[2001:db8:cafe::17]:4711;proTo=https;HoSt=baidu.com:7777",
			TrustProxies: []string{"2001:db8:cafe::17", "127.0.0.1"}, TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want: out{IP: "192.168.33.1", Port: 7777, Host: "baidu.com",
				Proto: routing.HTTPS}},
		{name: "xff with lower pri", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost:      "[2001:db8:1f70::999:de8:7648:6e8]:100",
			FF:              "for=192.168.33.1:834, fOr=[2001:db8:cafe::17]:4711;proTo=https;HoSt=baidu.com:7777",
			HeaderXFF:       "113.32.34.2,174.28.3.4:8888",
			HeaderXFH:       "google.com:2343",
			HeaderXFPRO:     "off",
			HeaderXFP:       "8769",
			TrustProxies:    []string{"174.28.3.4", "2001:db8:cafe::17", "127.0.0.1"},
			TrustHeadersSet: DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED,
			want:            out{IP: "113.32.34.2", Port: 8769, Host: "google.com", Proto: routing.HTTP},
		},
		{name: "trust header set", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost:      "[2001:db8:1f70::999:de8:7648:6e8]:100",
			FF:              "for=192.168.33.1:834, fOr=[2001:db8:cafe::17]:4711;proTo=https;HoSt=baidu.com:7777",
			HeaderXFF:       "113.32.34.2,174.28.3.4:8888",
			HeaderXFH:       "google.com:2343",
			HeaderXFPRO:     "off",
			HeaderXFP:       "8769",
			TrustProxies:    []string{"174.28.3.4", "2001:db8:cafe::17", "127.0.0.1"},
			TrustHeadersSet: (DefCfg.TrustedHeaderSet | routing.HEADER_FORWARDED) &^ (routing.HEADER_X_FORWARDED_HOST | routing.HEADER_X_FORWARDED_PROTO | routing.HEADER_X_FORWARDED_PORT | routing.HEADER_X_FORWARDED_FOR),
			want:            out{IP: "192.168.33.1", Port: 7777, Host: "baidu.com", Proto: routing.HTTPS},
		},
		{name: "zero trusted set", remoteIp: "127.0.0.1", remotePort: 3099,
			HeaderHost:      "[2001:db8:1f70::999:de8:7648:6e8]:100",
			FF:              "for=192.168.33.1:834, fOr=[2001:db8:cafe::17]:4711;proTo=https;HoSt=baidu.com:7777",
			HeaderXFF:       "113.32.34.2,174.28.3.4:8888",
			HeaderXFH:       "google.com:2343",
			HeaderXFPRO:     "off",
			HeaderXFP:       "8769",
			TrustProxies:    []string{"174.28.3.4", "2001:db8:cafe::17", "127.0.0.1"},
			TrustHeadersSet: 0,
			want:            out{IP: "127.0.0.1", Port: 100, Host: "2001:db8:1f70::999:de8:7648:6e8", Proto: routing.HTTP},
		},
	} {
		t.Run(in.name, func(t *testing.T) {
			assert.Equal(t, &in.want, getOut(&in))
		})
	}
}

func getOut(in *input) (r *out) {
	r = &out{}
	router = routing.New()
	router.Use(New(Config{TrustedHeaderSet: in.TrustHeadersSet, TrustProxies: in.TrustProxies}))
	router.Get("/", func(ctx *routing.Ctx) error {
		r.IP = unsafefn.BtoS(ctx.IP())
		r.Host = unsafefn.BtoS(ctx.Host())
		r.Proto = ctx.Proto()
		r.Port = ctx.Port()
		return nil
	})
	ctx := &fasthttp.RequestCtx{}
	req := &fasthttp.Request{}
	req.Header.SetHost(in.HeaderHost)
	req.Header.Set(routing.HeaderXForwardedHost, in.HeaderXFH)
	req.Header.Set(routing.HeaderXForwardedPort, in.HeaderXFP)
	req.Header.Set(routing.HeaderXForwardedProto, in.HeaderXFPRO)
	req.Header.Set(routing.HeaderXForwardedFor, in.HeaderXFF)
	req.Header.Set(routing.HeaderForwarded, in.FF)

	addr := &net.TCPAddr{IP: net.ParseIP(in.remoteIp), Port: in.remotePort}
	ctx.Init(req, addr, nil)
	router.HandleRequest(ctx)
	return
}
