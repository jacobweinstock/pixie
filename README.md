# pixie

```bash
sudo go run main.go file -ipxe-addr 192.168.2.225 -filename example/example.json -log-level debug -ipxe-script-addr http://192.168.2.225:8080 -proxy-dhcp-addr 192.168.2.225:67
```

- [] add http server support for serving `auto.ipxe` file
- [] support running both dhcp and proxyDHCP on same server
