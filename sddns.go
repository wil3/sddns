package sddns

import (
	"log"
	"net"
	"time"
	//"strconv"

	//"github.com/miekg/coredns/request"

	"fmt"
	"github.com/miekg/coredns/middleware"
	"github.com/miekg/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	//"strconv"
	"encoding/json"
	"net/http"
	"net/url"
)

type Sddns struct {
	Next              middleware.Handler
	controllerToken   string
	controllerAddress string

	rules map[string]*Rule
}

type Rule struct {
	ClientToken string
	Ipv4        string
	Ipv6        string
	Ttl         uint32 //Time to live of the DNS records
	Timeout     uint32 //How long the rule will stay in cache until the controller is re-queried
	//createTime int64 //Time in seconds when this rule was created
}

/**
 * DNS query
 */
func (s Sddns) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	log.Printf("Controller %s\n", s.controllerAddress)
	state := request.Request{W: w, Req: r}
	labels := dns.SplitDomainName(state.QName())
	log.Printf("Labels %v\n", labels)

	var rule Rule

	//TODO The query should be checked if it matchs, is this in Corefile?

	//The first label is the most subdomain
	token := labels[0]
	//TODO verify token MAC

	var ok bool
	var val *Rule
	if val, ok = s.rules[token]; ok {
		//Is the rule expired?
		//if time.Now().Unix() > (*val).createTime + int64((*val).Timeout) {
		if time.Now().Unix() > 0 {
			delete(s.rules, token)
			rule = askController(s.controllerAddress, token)
		} else {
			//Were good
			rule = (*val)
		}
	} else {
		//cache miss, ask controller
		rule = askController(s.controllerAddress, "")
	}

	sendResponse(rule, state)
	return dns.RcodeSuccess, nil
}

func askController(controllerAddress string, token string) Rule {
	u, err := url.ParseRequestURI(controllerAddress)
	if err != nil {
		log.Fatal("[Error] Parse %s\n", err)
	}
	u.Path = fmt.Sprintf("/rule/%s", token)

	log.Printf("Endpoint %s\n", u.String())

	rule := Rule{}
	err = getJson(u.String(), &rule)
	if err != nil {
		log.Printf("[Error] %s\n", err)
	}
	log.Printf("Controller %+v\n", rule)
	return rule
}

func sendResponse(rule Rule, state request.Request) {
	log.Println("Sending response")
	a := new(dns.Msg)
	a.SetReply(state.Req)
	a.Compress = true
	a.Authoritative = true

	var rr dns.RR

	switch state.Family() {
	case 1:
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: rule.Ttl}
		rr.(*dns.A).A = net.ParseIP(rule.Ipv4).To4()
	case 2:
		rr = new(dns.AAAA)
		rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass(), Ttl: rule.Ttl}
		rr.(*dns.AAAA).AAAA = net.ParseIP(rule.Ipv6)
	}
	a.Extra = []dns.RR{rr}
	state.SizeAndDo(a)
	state.W.WriteMsg(a)
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

// Name implements the Handler interface.
func (s Sddns) Name() string { return "sddns" }
