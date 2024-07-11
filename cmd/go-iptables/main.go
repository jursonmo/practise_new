package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
)

func main() {
	tables, err := iptables.New()
	if err != nil {
		panic(err)
	}
	chains, err := tables.ListChains("nat")
	if err != nil {
		panic(err)
	}
	for _, chain := range chains {
		fmt.Println("chain:", chain)
		rules, err := tables.List("nat", chain)
		if err != nil {
			panic(err)
		}
		for _, r := range rules {
			fmt.Println("rule:", r)
		}
	}
	err = tables.Insert("nat", "PREROUTING", 1, "-p", "udp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8080")
	if err != nil {
		panic(err)
	}
	err = tables.Append("nat", "PREROUTING", "-p", "udp", "--dport", "81", "-j", "REDIRECT", "--to-ports", "8081")
	if err != nil {
		panic(err)
	}
	ok, err := tables.Exists("nat", "PREROUTING", "-p", "udp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8080")
	if err != nil {
		panic(err)
	}
	fmt.Println("1111 exist:", ok)
	ok, err = tables.Exists("nat", "PREROUTING", "-p", "udp", "--dport", "90", "-j", "REDIRECT", "--to-ports", "8090")
	if err != nil {
		panic(err)
	}
	fmt.Println("22222 exist:", ok)
	err = tables.Delete("nat", "PREROUTING", "-p", "udp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8080")
	if err != nil {
		panic(err)
	}
	ok, err = tables.Exists("nat", "PREROUTING", "-p", "udp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8080")
	if err != nil {
		panic(err)
	}
	fmt.Println("33333 exist:", ok)
	err = tables.ClearChain("nat", "PREROUTING")
	if err != nil {
		panic(err)
	}
	ok, err = tables.Exists("nat", "PREROUTING", "-p", "udp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8008")
	if err != nil {
		panic(err)
	}
	fmt.Println("exist:", ok)
}
