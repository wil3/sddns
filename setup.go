package sddns

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/mholt/caddy"
)

func init() {
	caddy.RegisterPlugin("sddns", caddy.Plugin{
		ServerType: "dns",
		Action:     setupSddns,
	})
}

func setupSddns(c *caddy.Controller) error {
	sddns := Sddns{}
	for c.Next() {
		if c.Val() == "sddns" {
			for c.NextBlock() {
				what := c.Val()
				if !c.NextArg() {
					return c.ArgErr()
				}
				value := c.Val()
				switch what {
				case "controller_address":
					sddns.controllerAddress = value
				case "controller_token":
					sddns.controllerToken = value
				}
			}
		}
	}

	if (sddns.controllerToken == "") || (sddns.controllerAddress == "") {
		return middleware.Error("sddns", c.ArgErr())
	}

	dnsserver.GetConfig(c).AddMiddleware(func(next middleware.Handler) middleware.Handler {
		sddns.rules = make(map[string]*Rule)
		sddns.Next = next
		return sddns
	})

	return nil
}
