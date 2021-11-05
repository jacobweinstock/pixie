# pixie

```bash
sudo go run main.go file -ipxe-addr 192.168.2.225 -filename example/example.json -log-level debug -ipxe-script-addr http://192.168.2.225:8080 -proxy-dhcp-addr 192.168.2.225

sudo go run main.go file -ipxe-addr 192.168.1.34 -filename example/example.json -log-level debug -ipxe-script-addr http://192.168.2.225:8080 -proxy-dhcp-addr 192.168.1.34
```

- [] add http server support for serving `auto.ipxe` file
- [] support running both dhcp and proxyDHCP on same server
