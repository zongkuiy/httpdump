# httpdump
http/websocket dump tool, develop based on github.com/google/gopacket project

### Dependency

- libpcap

### Usage
```
Usage of httpdump:
  -assembly_debug_log
    	If true, the github.com/google/gopacket/tcpassembly library will log verbose debugging information (at least one line per packet)
  -assembly_memuse_log
    	If true, the github.com/google/gopacket/tcpassembly library will log information regarding its memory use every once in a while.
  -dev string
    	device watched, default any (default "any")
  -help
    	show usage
  -host string
    	host watched
  -port uint
    	port watched, default 80 (default 80)
  -pretty
    	show json/xml as pretty format, default true (default true)
 ```
