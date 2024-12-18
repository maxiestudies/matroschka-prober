package config

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/exaring/matroschka-prober/pkg/prober"
	"github.com/pkg/errors"
)

var (
	dfltBasePort = uint16(32768)
	dfltClass    = Class{
		Name: "BE",
		TOS:  0x00,
	}
	dfltTimeoutMS           = uint64(500)
	dfltListenAddress       = ":9517"
	dfltMeasurementLengthMS = uint64(1000)
	dfltPayloadSizeBytes    = uint64(0)
	dfltPPS                 = uint64(25)
	dfltSrcRange            = "169.254.0.0/16"
	dfltMetricsPath         = "/metrics"
)

// Config represents the configuration of matroschka-prober
type Config struct {
	// docgen:nodoc
	// this member is not configured on the yaml file
	Version string
	// description: |
	//   Path used to expose the metrics.
	MetricsPath *string `yaml:"metrcis_path"`
	// description: |
	//   Address used to listen for returned packets
	ListenAddress *string `yaml:"listen_address"`
	// description: |
	//   Port used to listen for returned packets
	BasePort *uint16 `yaml:"base_port"`
	// description: |
	//   Default configuration parameters
	Defaults *Defaults `yaml:"defaults"`
	// description: |
	//   Range of IP addresses used as a source for the package. Useful to add some variance in the parameters used to hash the packets in ECMP scenarios
	SrcRange *string `yaml:"src_range"`
	// description: |
	//   Class of services
	Classes []Class `yaml:"classes"`
	// description: |
	//   List of paths to probe
	Paths []Path `yaml:"paths"`
	// description: |
	//   List of routers used as explicit hops in the path.
	Routers []Router `yaml:"routers"`
}

// Defaults represents the default section of the config
type Defaults struct {
	// description: |
	//   Measurement interval expressed in milliseconds.
	//   IMPORTANT: If you are scraping the exposed metrics from /metrics, your scraping tool needs to scrape at least once in your defined interval.
	//   E.G if you define a measurement length of 1000ms, your scraping tool muss scrape at least 1/s, otherwise the data will be gone.
	MeasurementLengthMS *uint64 `yaml:"measurement_length_ms"`
	// description: |
	//   Optional size of the payload (default = 0).
	PayloadSizeBytes *uint64 `yaml:"payload_size_bytes"`
	// description: |
	//   Amount of probing packets that will be sent per second.
	PPS *uint64 `yaml:"pps"`
	// description: |
	//   Range of IP addresses used as a source for the package. Useful to add some variance in the parameters used to hash the packets in ECMP scenarios
	//   Defaults to 169.254.0.0/16
	SrcRange *string `yaml:"src_range"`
	// description: |
	//   Timeouts expressed in milliseconds
	TimeoutMS *uint64 `yaml:"timeout"`
	// description: |
	//  Source Interface
	SrcInterface *string `yaml:"src_interface"`
}

// Class reperesnets a traffic class in the config file
type Class struct {
	// description: |
	//   Name of the traffic class.
	Name string `yaml:"name"`
	// description: |
	//    Type of Service assigned to the class.
	TOS uint8 `yaml:"tos"`
}

// Path represents a path to be probed
type Path struct {
	// description: |
	//   Name for the path.
	Name string `yaml:"name"`
	// description: |
	//   List of hops to probe.
	Hops []string `yaml:"hops"`
	// description: |
	//   Measurement interval expressed in milliseconds.
	MeasurementLengthMS *uint64 `yaml:"measurement_length_ms"`
	// description: |
	//   Payload size expressed in Bytes.
	PayloadSizeBytes *uint64 `yaml:"payload_size_bytes"`
	// description: |
	//   Amount of probing packets that will be sent per second.
	PPS *uint64 `yaml:"pps"`
	// description: |
	//   Timeout expressed in milliseconds.
	TimeoutMS *uint64 `yaml:"timeout"`
}

// Router represents a router used a an explicit hop in a path
type Router struct {
	// description: |
	//   Name of the router.
	Name string `yaml:"name"`
	// description: |
	//   Destination range of IP addresses.
	DstRange string `yaml:"dst_range"`
	// description: |
	//   Range of source ip addresses.
	SrcRange string `yaml:"src_range"`
}

// Validate validates a configuration
func (c *Config) Validate() error {
	err := c.validatePaths()
	if err != nil {
		return fmt.Errorf("Path validation failed: %v", err)
	}

	err = c.validateRouters()
	if err != nil {
		return fmt.Errorf("Router validation failed: %v", err)
	}

	return nil
}

func (c *Config) validatePaths() error {
	for i := range c.Paths {
		for j := range c.Paths[i].Hops {
			if !c.routerExists(c.Paths[i].Hops[j]) {
				return fmt.Errorf("Router %q of path %q does not exist", c.Paths[i].Hops[j], c.Paths[i].Name)
			}
		}
	}

	return nil
}

func (c *Config) routerExists(needle string) bool {
	for i := range c.Routers {
		if c.Routers[i].Name == needle {
			return true
		}
	}

	return false
}

func (c *Config) validateRouters() error {
	for i := range c.Routers {
		_, _, err := net.ParseCIDR(c.Routers[i].DstRange)
		if err != nil {
			return fmt.Errorf("Unable to parse dst IP range for router %q: %v", c.Routers[i].Name, err)
		}
	}

	return nil
}

// ApplyDefaults applies default settings if they are missing from loaded config.
func (c *Config) ApplyDefaults() {
	if c.Defaults == nil {
		c.Defaults = &Defaults{}
	}
	c.Defaults.applyDefaults()

	if c.SrcRange == nil {
		c.SrcRange = c.Defaults.SrcRange
	}

	if c.MetricsPath == nil {
		c.MetricsPath = &dfltMetricsPath
	}

	if c.ListenAddress == nil {
		c.ListenAddress = &dfltListenAddress
	}

	if c.BasePort == nil {
		c.BasePort = &dfltBasePort
	}

	for i := range c.Paths {
		c.Paths[i].applyDefaults(c.Defaults)
	}

	for i := range c.Routers {
		c.Routers[i].applyDefaults(c.Defaults)
	}

	if c.Classes == nil {
		c.Classes = []Class{
			dfltClass,
		}
	}
}

func (r *Router) applyDefaults(d *Defaults) {
	if r.SrcRange == "" {
		r.SrcRange = *d.SrcRange
	}
}

func (p *Path) applyDefaults(d *Defaults) {
	if p.MeasurementLengthMS == nil {
		p.MeasurementLengthMS = d.MeasurementLengthMS
	}

	if p.PayloadSizeBytes == nil {
		p.PayloadSizeBytes = d.PayloadSizeBytes
	}

	if p.PPS == nil {
		p.PPS = d.PPS
	}

	if p.TimeoutMS == nil {
		p.TimeoutMS = d.TimeoutMS
	}
}

func (d *Defaults) applyDefaults() {
	if d.MeasurementLengthMS == nil {
		d.MeasurementLengthMS = &dfltMeasurementLengthMS
	}

	if d.PayloadSizeBytes == nil {
		d.PayloadSizeBytes = &dfltPayloadSizeBytes
	}

	if d.PPS == nil {
		d.PPS = &dfltPPS
	}

	if d.SrcRange == nil {
		d.SrcRange = &dfltSrcRange
	}

	if d.TimeoutMS == nil {
		d.TimeoutMS = &dfltTimeoutMS
	}
}

// GetConfiguredSrcAddr gets an IPv4 address of the configured src interface
func (c *Config) GetConfiguredSrcAddr() (net.IP, error) {
	if c.Defaults.SrcInterface == nil {
		return nil, nil
	}

	return GetInterfaceAddr(*c.Defaults.SrcInterface)
}

// GetInterfaceAddr gets an interface first IPv4 address
func GetInterfaceAddr(ifName string) (net.IP, error) {
	ifa, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get interface")
	}

	addrs, err := ifa.Addrs()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get addresses")
	}

	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err != nil {
			continue
		}

		if ip.To4() == nil {
			continue
		}

		return ip, nil
	}

	return nil, nil
}

// PathToProberHops generates prober hops
func (c *Config) PathToProberHops(pathCfg Path) []prober.Hop {
	res := make([]prober.Hop, 0)

	for i := range pathCfg.Hops {
		for j := range c.Routers {
			if pathCfg.Hops[i] != c.Routers[j].Name {
				continue
			}

			h := prober.Hop{
				Name:     c.Routers[j].Name,
				DstRange: GenerateAddrs(c.Routers[j].DstRange),
				SrcRange: GenerateAddrs(c.Routers[j].SrcRange),
			}
			res = append(res, h)
		}
	}

	return res
}

// GenerateAddrs returns a list of all IPs in addrRange
func GenerateAddrs(addrRange string) []net.IP {
	_, n, err := net.ParseCIDR(addrRange)
	if err != nil {
		panic(err)
	}

	baseAddr := getCIDRBase(*n)
	c := maskAddrCount(*n)
	ret := make([]net.IP, c)

	for i := uint32(0); i < c; i++ {
		ret[i] = net.IP(uint32Byte(baseAddr + i%c))
	}

	return ret
}

func getCIDRBase(n net.IPNet) uint32 {
	return uint32b(n.IP)
}

func uint32b(data []byte) (ret uint32) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &ret)
	return
}

func getNthAddr(n net.IPNet, i uint32) net.IP {
	baseAddr := getCIDRBase(n)
	c := maskAddrCount(n)
	return net.IP(uint32Byte(baseAddr + i%c))
}

func maskAddrCount(n net.IPNet) uint32 {
	ones, bits := n.Mask.Size()
	if ones == bits {
		return 1
	}

	x := uint32(1)
	for i := ones; i < bits; i++ {
		x = x * 2
	}
	return x
}

func uint32Byte(data uint32) (ret []byte) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, data)
	return buf.Bytes()
}
