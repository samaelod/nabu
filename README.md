# Nabu

**Nabu** is a configurable network traffic emulator for stress testing, QA, and dynamic analysis. Load a PCAP file or write a Lua script, then replay captured network flows against live servers or use as a traffic generator.

## Features

- **PCAP Import**: Parse `.pcap` files to automatically generate emulation scenarios
- **Lua Scripting**: Define custom network topologies and traffic patterns in Lua
- **Live Emulation**: Replay captured traffic against real servers or as a traffic generator
- **Real-time TUI**: Monitor endpoint status and live logs while emulating
- **Stress Testing**: Run multiple endpoints simultaneously for load testing
- **Simple & Extensible**: Clean Lua API for custom scenarios

## Installation

### Prerequisites

- Go 1.25+
- `libpcap` dev headers (usually `libpcap-dev` or `libpcap-devel` on Linux)

### Build

```bash
git clone https://github.com/samaelod/nabu.git
cd nabu
make build
```

Run with:
```bash
make run
```

Or directly:
```bash
./bin/nabu
```

## Usage

1. **Select Source**: Choose **PCAP File** or **Lua Script** from the main menu
2. **Browse**: Use the file browser to locate your `.pcap` or `.lua` file
3. **Inspect**: View detected endpoints and message flows
4. **Run**: Press `r` to run the selected endpoint, `s` to stop

## Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate menu/list |
| `Enter` | Select |
| `r` | Run selected endpoint |
| `s` | Stop selected endpoint |
| `e` | Edit config (when endpoint focused) / Open logs in editor (when logs focused) |
| `u` | Update config from file |
| `<tab>` | Switch focus between panels |
| `g` | Go to top of logs |
| `G` | Go to bottom of logs |
| `q` | Quit |

## Configuration

Nabu uses Lua for configuration. Here's the full structure:

### Globals

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `protocol` | string | "tcp" | Network protocol (currently TCP only) |
| `play_mode` | string | "pcap" | Playback mode |
| `timeout` | int | 5000 | Connection timeout (ms) |
| `delay` | int | 100 | Delay between messages (ms) |
| `log_lines` | int | 1000 | In-memory log buffer size |

### Endpoints

```lua
config.endpoints = {
    { id = 0, kind = "client", address = "127.0.0.1", port = 8080 },
    { id = 1, kind = "server", address = "127.0.0.1", port = 8081 },
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Unique endpoint identifier |
| `kind` | string | "client" or "server" |
| `address` | string | IP address |
| `port` | int | Port number |

### Messages

```lua
config.messages = {
    { from = 0, to = 1, kind = "syn", value = "", t_delta = 0 },
    { from = 1, to = 0, kind = "syn-ack", value = "", t_delta = 10 },
    { from = 0, to = 1, kind = "ack", value = "", t_delta = 5 },
    { from = 0, to = 1, kind = "data", value = "48656c6c6f", t_delta = 100 },
}
```

| Field | Type | Description |
|-------|------|-------------|
| `from` | int | Source endpoint ID |
| `to` | int | Destination endpoint ID |
| `kind` | string | Message type: "syn", "syn-ack", "ack", "data", "fin" |
| `value` | string | Hex-encoded payload |
| `t_delta` | int | Delay before this message (ms) |

## Use Cases

- **Stress Testing**: Run multiple clients to test server capacity
- **Regression Testing**: Replay captured traffic to verify fixes
- **Development**: Simulate network conditions for local development
- **Security Testing**: Analyze application behavior with real traffic

## Limitations

- **Protocols**: Currently TCP only (UDP, HTTP coming soon)
- **Features**: Some advanced features like configurable timestamps, encoding options, and packet modification planned for future releases

## Roadmap

- [ ] UDP support
- [ ] HTTP/HTTPS emulation
- [ ] Configurable packet timestamps
- [ ] Packet encoding options
- [ ] Packet modification/transformation

## License

MIT
