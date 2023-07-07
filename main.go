package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/andrewslotin/llog"
	"github.com/endocrimes/keylight-go"
)

var (
	Version = "0.0.1"
	args    struct {
		Brightness  uint
		Temperature uint
		Timeout     time.Duration
		Verbose     bool
		Debug       bool
	}
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "keylightctl v%s\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.UintVar(&args.Brightness, "b", 0, "Brightness (in percent)")
	flag.UintVar(&args.Temperature, "k", 0, "Temperature (in Kelvin)")
	flag.DurationVar(&args.Timeout, "t", 10*time.Second, "Discovery timeout")
	flag.BoolVar(&args.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&args.Debug, "vv", false, "Debug output")
	flag.Parse()

	logLevel := llog.WarnLevel
	if args.Verbose {
		logLevel = llog.InfoLevel
	}

	if args.Debug {
		logLevel = llog.DebugLevel
	}

	log.SetFlags(0)
	log.SetOutput(llog.NewWriter(os.Stderr, logLevel))

	discovery, err := keylight.NewDiscovery()
	if err != nil {
		log.Fatalf("failed to create discovery: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		dCtx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		defer cancel()

		log.Printf("debug: running discovery (timeout %s)...", args.Timeout)
		if err := discovery.Run(dCtx); err != nil {
			log.Fatalf("failed to run discovery: %s", err)
		}
	}(ctx)

	light, ok := <-discovery.ResultsCh()
	cancel()

	if !ok {
		fmt.Fprintln(os.Stderr, "no lights found")
		os.Exit(1)
	}

	log.Println("debug: found light ", light.Name, light.DNSAddr)

	opts, err := light.FetchLightGroup(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch options: %s", err)
	}

	for i, light := range opts.Lights {
		log.Printf("light %d before: %d%% %dK", i+1, light.Brightness, convertTemp(light.Temperature))
	}

	opts = opts.Copy()
	for _, light := range opts.Lights {
		if args.Brightness > 0 {
			light.Brightness = int(args.Brightness)
		}
		if args.Temperature > 0 {
			light.Temperature = 1000000 / int(args.Temperature)
		}
	}

	if _, err := light.UpdateLightGroup(context.Background(), opts); err != nil {
		log.Fatalf("failed to update light group: %s", err)
	}

	opts, err = light.FetchLightGroup(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch options: %s", err)
	}

	for i, light := range opts.Lights {
		log.Printf("light %d after: %d%% %dK", i+1, light.Brightness, convertTemp(light.Temperature))
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
