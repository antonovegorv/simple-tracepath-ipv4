package config

type Config struct {
	Hostname   string
	Timeout    int
	MaxTTL     int
	PacketSize int
}

func New(hostname string, timeout, maxTTL, packetSize int) *Config {
	return &Config{
		Hostname:   hostname,
		Timeout:    timeout,
		MaxTTL:     maxTTL,
		PacketSize: packetSize,
	}
}
