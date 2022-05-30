# DNSMAP

proxying dns server, which manages NAT mappings for all of resolved domains

example flow:
```
; client -> dnsmap question
example.com. IN A

; dnsmap -> upstream question
example.com. IN A

; dnsmap <- upstream answer
example.com. 1500 IN A 192.0.2.1

; here dnsmap looks up 192.0.2.1 in internal cache
; if it's not present, new address is allocated from remap pool
; and both NFTables map and internal cache are populated
; remap 192.0.2.1 as 10.222.0.1 established
; nft add element nat dnsmap '{ 10.222.0.1 : 192.0.2.1 }'

; client <- dnsmap
example.com. 1500 IN A 10.222.0.1

; now when client connects to 10.222.0.1, nftables NATs
; those packets to original IP addresses
```

# TODO
clean up and expire mappings by TTL and usage in nftables

# License
MIT
