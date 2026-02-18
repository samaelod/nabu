package pcapreader

import (
	"encoding/hex"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	"github.com/samaelod/nabu/types"
)

var ipPortRegex = regexp.MustCompile(`^([\d.]+|[a-fA-F0-9:]+):(\d+)`)

type packetSource interface {
	LinkType() layers.LinkType
	ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error)
}

type packetDataSource struct {
	src      packetSource
	linkType layers.LinkType
}

func (p *packetDataSource) LinkType() layers.LinkType {
	return p.linkType
}

func (p *packetDataSource) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return p.src.ReadPacketData()
}

func detectFormat(path string) (format string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 8 bytes to check magic
	header := make([]byte, 8)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return "pcap", nil // Default to pcap
	}

	// Check for pcapng magic (Section Header Block)
	// PCAPNG starts with 0x0A0D0D0A (4 bytes)
	if n >= 4 {
		magic := uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16 | uint32(header[3])<<24
		if magic == 0x0A0D0D0A {
			return "pcapng", nil
		}
	}

	// Check for classic pcap magic
	// pcap: 0xA1B2C3D4 or 0xD4C3B2A1 (little/big endian)
	if n >= 4 {
		magic := uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16 | uint32(header[3])<<24
		if magic == 0xA1B2C3D4 || magic == 0xD4C3B2A1 || magic == 0xA1B23C4D || magic == 0x4D3CB2A1 {
			return "pcap", nil
		}
	}

	// Default to pcap
	return "pcap", nil
}

func openPacketSource(path string) (packetSource, error) {
	format, err := detectFormat(path)
	if err != nil {
		return nil, err
	}

	if format == "pcapng" {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		reader, err := pcapgo.NewNgReader(file, pcapgo.DefaultNgReaderOptions)
		if err != nil {
			file.Close()
			return nil, err
		}
		return &pcapngSource{reader: reader, file: file}, nil
	}

	// Classic pcap
	handle, err := pcap.OpenOffline(path)
	if err != nil {
		return nil, err
	}
	return &pcapSource{handle: handle}, nil
}

type pcapSource struct {
	handle *pcap.Handle
}

func (p *pcapSource) LinkType() layers.LinkType {
	return p.handle.LinkType()
}

func (p *pcapSource) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = p.handle.ReadPacketData()
	return
}

type pcapngSource struct {
	reader *pcapgo.NgReader
	file   *os.File
}

func (p *pcapngSource) LinkType() layers.LinkType {
	return p.reader.LinkType()
}

func (p *pcapngSource) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return p.reader.ReadPacketData()
}

func ReadPCAP(path string) (*types.Config, error) {
	source, err := openPacketSource(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ps, ok := source.(*pcapngSource); ok {
			ps.file.Close()
		} else if ps, ok := source.(*pcapSource); ok {
			ps.handle.Close()
		}
	}()

	cfg := &types.Config{
		Globals: types.Globals{
			Protocol: "tcp",
			PlayMode: "pcap",
			Timeout:  5000,
			Delay:    100,
		},
	}

	endpointMap := make(map[string]int)
	var nextEndpointID int

	var prevTime time.Time

	ds := &packetDataSource{src: source, linkType: source.LinkType()}
	packetSrc := gopacket.NewPacketSource(ds, ds.LinkType())

	for packet := range packetSrc.Packets() {

		net := packet.NetworkLayer()
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if net == nil || tcpLayer == nil {
			continue
		}

		tcp := tcpLayer.(*layers.TCP)
		payload := tcp.Payload

		// Determine Kind based on flags
		var kind string
		if tcp.SYN && !tcp.ACK {
			kind = "syn"
		} else if tcp.SYN && tcp.ACK {
			kind = "syn-ack"
		} else if tcp.FIN {
			kind = "fin"
		} else if tcp.RST {
			kind = "rst"
		} else if len(payload) > 0 {
			kind = "data"
		} else if tcp.ACK {
			kind = "ack"
		}

		// Skip only if it's a pure ack with no payload AND we assume we only want significant events?
		// User wants connection attempts (SYN, SYN-ACK) which usually have no payload.
		// So we must NOT skip empty payload if it has interesting flags.
		// We skip if it's empty AND not a control packet we care about?
		// For now, let's keep all. Or maybe skip pure ACKs if they are too noisy?
		// User asked for "ack, sin-ack", so we include them.

		if len(payload) == 0 && kind == "" {
			continue
		}

		// Extract IP addresses properly based on layer type
		var srcIP, dstIP string
		if ipv4, ok := net.(*layers.IPv4); ok {
			srcIP = ipv4.SrcIP.String()
			dstIP = ipv4.DstIP.String()
		} else if ipv6, ok := net.(*layers.IPv6); ok {
			srcIP = ipv6.SrcIP.String()
			dstIP = ipv6.DstIP.String()
		} else {
			// Fallback to NetworkFlow().Src().String()
			srcIP = net.NetworkFlow().Src().String()
			dstIP = net.NetworkFlow().Dst().String()
		}

		srcKey := srcIP + ":" + tcp.SrcPort.String()
		dstKey := dstIP + ":" + tcp.DstPort.String()

		srcID := getOrCreateEndpoint(srcKey, "client", &cfg.Endpoints, endpointMap, &nextEndpointID)
		dstID := getOrCreateEndpoint(dstKey, "server", &cfg.Endpoints, endpointMap, &nextEndpointID)

		delta := 0
		ts := packet.Metadata().Timestamp
		if !prevTime.IsZero() {
			delta = int(ts.Sub(prevTime).Milliseconds())
		}
		prevTime = ts

		msg := types.Message{
			From:   srcID,
			To:     dstID,
			Kind:   kind,
			Value:  hex.EncodeToString(payload),
			TDelta: delta,
		}

		cfg.Messages = append(cfg.Messages, msg)
	}

	return cfg, nil
}

func getOrCreateEndpoint(
	key string,
	kind string,
	endpoints *[]types.Endpoint,
	index map[string]int,
	nextID *int,
) int {

	if id, ok := index[key]; ok {
		return id
	}

	id := *nextID
	*nextID++

	index[key] = id

	// Use regex to extract IP and port from key like "10.1.0.1:5001"
	matches := ipPortRegex.FindStringSubmatch(key)
	var address string
	var port int
	if matches != nil {
		address = matches[1]
		if p, err := strconv.Atoi(matches[2]); err != nil {
			log.Printf("Warning: failed to parse port %q: %v", matches[2], err)
		} else {
			port = p
		}
	} else {
		// Fallback: use whole key as address
		address = key
		port = 0
	}

	*endpoints = append(*endpoints, types.Endpoint{
		ID:      id,
		Kind:    kind,
		Address: address,
		Port:    port,
	})

	return id
}
