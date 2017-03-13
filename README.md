# SDDNS

SDDNS is middleware for CoreDNS

1. Clone CoreDNS
2. Add SDDNS to imports in  core/coredns.go
"github.com/wil3/sddns"

3.In core/dnsserver/directives.go
add `sddns` to directives array. 

4. Add SDDNS to CoreDNS's configuration file, `Corefile`
