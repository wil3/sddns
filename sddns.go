package sddns

import (
	"net"
	"time"
	//"strconv"

	//"github.com/miekg/coredns/request"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"fmt"
	"github.com/miekg/coredns/middleware"
	"github.com/miekg/coredns/request"
	"strconv"
)
type Sddns struct {

	Next middleware.Handler
	// Index 0 would be TLD
	tokenLabelIndex uint8
	controllerToken string
	controllerAddress string

	rules map[string]*Rule
}

type Rule struct {
	clientToken string
	ipv4 string
	ipv6 string
	ttl int //Time to live of the DNS records
	expire int //How long the rule will stay in cache until the controller is re-queried
}
/**
 * Request for Controller
 */
func (s Sddns) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	labels := dns.SplitDomainName(state.QName())
	fmt.Print("Labels " + labels)
	if len(labels) < s.tokenLabelIndex {
		return middleware.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}
	token := labels[s.tokenLabelIndex]

	if rule, ok := s.rules[token]; ok {

		//Is the rule expired?
		if time.Now().Unix() > rule.expire {
			//Ask controller
			delete(s.rules, token)

		} else {
			//Rule is good
			sendResponse(rule, state)
		}
	} else {
		//cache miss, ask controller
	}

	return dns.RcodeSuccess, nil
}
func sendResponse(rule Rule, state request.Request) {
	a := new(dns.Msg)
	a.SetReply(state.Req)
	a.Compress = true
	a.Authoritative = true

	var rr dns.RR

	switch state.Family() {
	case 1:
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: rule.ttl}
		rr.(*dns.A).A = net.ParseIP(rule.ipv4).To4()
	case 2:
		rr = new(dns.AAAA)
		rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass(), Ttl: rule.ttl}
		rr.(*dns.AAAA).AAAA = net.ParseIP(rule.ipv6)
	}

	state.SizeAndDo(a)
	state.W.WriteMsg(a)
}
// Name implements the Handler interface.
func (s Sddns) Name() string { return "sddns" }
