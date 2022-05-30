package main

type confLog struct {
	Query  bool `json:"query"`
	Answer bool `json:"answer"`
}

type confUpstream struct {
	Addr string `json:"address"`
	Net  string `json:"net"`
}

type confListen struct {
	Addr string `json:"address"`
	UDP  bool   `json:"udp"`
	TCP  bool   `json:"tcp"`
}

type confRemapNFT struct {
	Table string `json:"table"`
	Map   string `json:"map"`
	Clear bool   `json:"clear"`
}

type confRemap struct {
	Range string       `json:"range"`
	NFT   confRemapNFT `json:"nftables"`
}

type confRoot struct {
	Log      confLog      `json:"log"`
	Upstream confUpstream `json:"upstream"`
	Listen   confListen   `json:"listen"`
	Remap    confRemap    `json:"remap"`
}

var defaultConf = confRoot{
	Log: confLog{
		Query:  false,
		Answer: false,
	},
	Upstream: confUpstream{
		Addr: "8.8.8.8:53",
		Net:  "tcp",
	},
	Listen: confListen{
		Addr: "127.0.0.4:53",
		TCP:  true,
		UDP:  true,
	},
	Remap: confRemap{
		Range: "10.222.0.0/20",
		NFT: confRemapNFT{
			Table: "nat",
			Map:   "dnsmap",
			Clear: false,
		},
	},
}
