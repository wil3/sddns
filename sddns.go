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
	"errors"
	"net/http"
	"net/url"
)

type Sddns struct {
	Next              middleware.Handler
	controllerToken   string
	controllerAddress string
	rules             map[string]*Rule
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

	log.Println("BEGIN")
	state := request.Request{W: w, Req: r}

	labels := dns.SplitDomainName(state.QName())
	log.Printf("Labels %v\n", labels)

	var rule Rule

	if val, ok := s.rules[state.QName()]; ok {
		//Were good, already have it
		log.Println("Rule already exists in cache, returning")
		rule = (*val)
		sendResponse(rule, state)
	} else {
		//cache miss, ask controller
		rule, err := askController(s.controllerAddress, state.QName())
		if err != nil {
			return dns.RcodeNameError, err
		}
		s.rules[state.QName()] = &rule
		sendResponse(rule, state)
	}

	log.Println("END")
	return dns.RcodeSuccess, nil
}

func askController(controllerAddress string, qname string) (Rule, error) {
	log.Printf("Qname is \"%s\"", qname)
	u, err := url.ParseRequestURI(controllerAddress)
	if err != nil {
		log.Fatal("[Error] Parse %s\n", err)
	}
	u.Path = fmt.Sprintf("/rule/%s", qname)

	log.Printf("Sending request to controller %s\n", u.String())

	rule := Rule{}
	err = getJson(u.String(), &rule)
	if err != nil {
		log.Printf("[Error] %s\n", err)
		return rule, err
	}
	log.Printf("Controller response: \"%+v\"", rule)
	return rule, nil
}

func sendResponse(rule Rule, state request.Request) {
	log.Println("Sending response")
	a := new(dns.Msg)
	a.SetReply(state.Req)
	a.Compress = true
	a.Authoritative = true

	var rr dns.RR

	log.Println("State family %d", state.Family())
	switch state.Family() {
	case 1:
		log.Println("IPv4")
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: rule.Ttl}
		rr.(*dns.A).A = net.ParseIP(rule.Ipv4).To4()
	case 2:
		log.Println("IPv6*")
		//rr = new(dns.AAAA)
		//rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass(), Ttl: rule.Ttl}
		//rr.(*dns.AAAA).AAAA = net.ParseIP(rule.Ipv6)
		rr = new(dns.SOA)
		rr.(*dns.SOA).Hdr = dns.RR_Header{Name: "token.wfk.io.", Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 0}
		rr.(*dns.SOA).Ns = "ns1.token.wfk.io"
		rr.(*dns.SOA).Mbox = "hostmaster.token.wfk.io.token.wfk.io"
		rr.(*dns.SOA).Serial = 2017041200
		rr.(*dns.SOA).Refresh = 1200
		rr.(*dns.SOA).Retry = 900
		rr.(*dns.SOA).Expire = 1209600
		rr.(*dns.SOA).Minttl = 3600
	}
	a.Answer = []dns.RR{rr}
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

	if r.StatusCode == http.StatusNotFound {
		return errors.New("Not found")
	}

	return json.NewDecoder(r.Body).Decode(target)
}

// Name implements the Handler interface.
func (s Sddns) Name() string { return "sddns" }
