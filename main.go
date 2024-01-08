package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/andrewslotin/llog"
	"github.com/endocrimes/keylight-go"
)

const (
	tempSetFlag uint8 = 1 << iota
	brightnessSetFlag
)

var (
	Version = "0.0.2"
	args    struct {
		ProvidedSettings uint8
		Brightness       uint
		Temperature      uint
		Timeout          time.Duration
		Verbose          bool
		Debug            bool
	}

	ErrNoLights = fmt.Errorf("no lights found")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "keylightctl v%s\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<host:port>[, ...]]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.UintVar(&args.Brightness, "b", 0, "Brightness (in percent)")
	flag.UintVar(&args.Temperature, "k", 0, "Temperature (in Kelvin)")
	flag.DurationVar(&args.Timeout, "t", 10*time.Second, "Discovery timeout")
	flag.BoolVar(&args.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&args.Debug, "vv", false, "Debug output")
	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "b":
			log.Printf("brightness is set")
			args.ProvidedSettings |= brightnessSetFlag
		case "k":
			log.Printf("temp is set")
			args.ProvidedSettings |= tempSetFlag
		}
	})

	logLevel := llog.WarnLevel
	if args.Verbose {
		logLevel = llog.InfoLevel
	}

	if args.Debug {
		logLevel = llog.DebugLevel
	}

	log.SetFlags(0)
	log.SetOutput(llog.NewWriter(os.Stderr, logLevel))

	var devices []*keylight.Device
	for _, arg := range flag.Args() {
		host, portStr, err := net.SplitHostPort(arg)
		if err != nil {
			log.Fatalf("failed to parse %q: %s", arg, err)
		}

		port, err := net.LookupPort("tcp", portStr)
		if err != nil {
			log.Fatalf("failed to lookup port %q: %s", portStr, err)
		}

		devices = append(devices, &keylight.Device{
			Name:    "User-specified device",
			DNSAddr: host,
			Port:    port,
		})
	}

	if len(devices) == 0 {
		log.Printf("debug: no devices specified via the command-line, using the discovery")
		ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		device, err := discoverDevice(ctx)
		if err != nil {
			log.Fatalf("failed to discover device: %s", err)
		}
		cancel()

		devices = append(devices, device)
	}

	var config []configurator
	if args.ProvidedSettings&brightnessSetFlag != 0 {
		config = append(config, brightnessConfigurator(args.Brightness))
	}

	if args.ProvidedSettings&tempSetFlag != 0 {
		config = append(config, temperatureConfigurator(args.Temperature))
	}

	for _, device := range devices {
		log.Println("debug: found light ", device.Name, device.DNSAddr)
		if err := updateDeviceSettings(context.Background(), device, config); err != nil {
			log.Fatalf("failed to update device %s settings: %s", device.DNSAddr, err)
		}
	}
}

func discoverDevice(ctx context.Context) (*keylight.Device, error) {
	discovery, err := keylight.NewDiscovery()
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		log.Printf("debug: running discovery (timeout %s)...", args.Timeout)
		if err := discovery.Run(ctx); err != nil {
			log.Fatalf("failed to run discovery: %s", err)
		}
	}()

	light, ok := <-discovery.ResultsCh()
	cancel()

	if !ok {
		return nil, ErrNoLights
	}

	return light, nil
}

func updateDeviceSettings(ctx context.Context, device *keylight.Device, config []configurator) error {
	opts, err := device.FetchLightGroup(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch options: %w", err)
	}

	for i, light := range opts.Lights {
		log.Printf("%s light #%d (%d) before: %d%% %dK", device.DNSAddr, i+1, light.On, light.Brightness, convertTemp(light.Temperature))
	}

	opts = opts.Copy()
	for _, light := range opts.Lights {
		for _, c := range config {
			c(light)
		}
	}

	if _, err := device.UpdateLightGroup(ctx, opts); err != nil {
		return fmt.Errorf("failed to update light group: %w", err)
	}

	opts, err = device.FetchLightGroup(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch options: %w", err)
	}

	for i, light := range opts.Lights {
		log.Printf("%s light #%d (%d) after: %d%% %dK", device.DNSAddr, i+1, light.On, light.Brightness, convertTemp(light.Temperature))
	}

	return nil
}

type configurator func(l *keylight.Light)

func brightnessConfigurator(brightness uint) configurator {
	return func(l *keylight.Light) {
		l.Brightness = int(brightness)
	}
}

func temperatureConfigurator(temperature uint) configurator {
	return func(l *keylight.Light) {
		l.Temperature = int(1000000 / temperature)
	}
}

// The 'Control Center' UI only allows color temperature changes from 2900K - 7000K in 50K increments.
// To convert the values used by the API to Kelvin (like it is displayed in the Elgato application), you
// simply have to divide 1.000.000 by the given value and round the result to the nearest number divisable by 50.
// When a value higher than the minimum or maximum (143 or 344) is sent, the value will be set to the closer extreme.
// To convert values from Kelvin back to the API format, just divide 1.000.000 by the amount of Kelvin and round to the
// closest natural number.
//
// Source: https://github.com/adamesch/elgato-key-light-api/blob/master/resources/lights/README.md
func convertTemp(elgatoUnits int) int {
	return int(math.Round(1000000.0/float64(elgatoUnits*50))) * 50
}
