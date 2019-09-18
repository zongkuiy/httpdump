package main

import (
	"flag"
	"github.com/google/gopacket/tcpassembly"
	"log"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	help         bool
	device       string        = "any"
	host         string        = ""
	snapshotLeng int32         = 65535
	promiscuous  bool          = false
	timeout      time.Duration = 30 * time.Second
	port                       = uint(80)

	prettyJson bool = true
)

func init() {
	flag.BoolVar(&help, "help", false, "show usage")
	flag.BoolVar(&prettyJson, "pretty", true, "show json/xml as pretty format, default true")
	flag.StringVar(&device, "dev", "any", "device watched, default any")
	flag.StringVar(&host, "host", "", "host watched")
	flag.UintVar(&port, "port", 80, "port watched, default 80")

}
func main() {

	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if len(device) == 0 {
		log.Println("device must be given")
		flag.Usage()
		return
	}

	startProcessing()

}

func startProcessing() {

	filterStr := "tcp"

	if len(host) > 0 {
		filterStr += " and host " + host
	}

	filterStr += " port " + strconv.Itoa(int(port))

	filter := flag.String("f", filterStr, "BPF filter for pcap")

	handle, err := pcap.OpenLive(device, snapshotLeng, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("start capture with filter value [" + filterStr + "] on device [" + device + "]")
	defer handle.Close()

	handle.SetBPFFilter(*filter)

	streamFactory := &httpParser{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		// Process packet here
		tcpLayer := packet.TransportLayer().(*layers.TCP)
		assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcpLayer, packet.Metadata().Timestamp)
	}
}
