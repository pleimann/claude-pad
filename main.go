package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/pleimann/camel-pad/internal/action"
	"github.com/pleimann/camel-pad/internal/config"
	"github.com/pleimann/camel-pad/internal/configpush"
	"github.com/pleimann/camel-pad/internal/display"
	"github.com/pleimann/camel-pad/internal/gesture"
	"github.com/pleimann/camel-pad/internal/hid"
	"github.com/pleimann/camel-pad/internal/pty"
	"github.com/pleimann/camel-pad/internal/ui"
)

const Version = "0.1.0"

func main() {
	// Check for subcommands first
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list-devices":
			runListDevices()
			return
		case "set-device", "select-device":
			runSetDevice(os.Args[2:])
			return
		case "config-push":
			runConfigPush(os.Args[2:])
			return
		case "help", "-h", "--help":
			printUsage()
			os.Exit(0)
		}
	}

	// Main command flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	version := flag.Bool("version", false, "print version and exit")

	flag.Usage = printUsage
	flag.Parse()

	if *version {
		ui.PrintVersion(Version)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *verbose {
		log.Printf("Loaded configuration from %s", *configPath)
		log.Printf("Device: VendorID=0x%04X, ProductID=0x%04X",
			cfg.Device.VendorID, cfg.Device.ProductID)
		log.Printf("TUI command: %s %v", cfg.TUI.Command, cfg.TUI.Args)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	app, err := newApp(cfg, *verbose)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	go func() {
		<-sigChan
		if *verbose {
			log.Println("Received shutdown signal")
		}
		cancel()
	}()

	if err := app.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("Application error: %v", err)
	}

	if *verbose {
		log.Println("Shutdown complete")
	}
}

func printUsage() {
	ui.PrintUsage(Version)
}

// runListDevices handles the list-devices subcommand
func runListDevices() {
	devices, err := hid.ListDevices()
	if err != nil {
		ui.PrintFatalError("Failed to list devices", err.Error())
		os.Exit(1)
	}
	uiDevices := make([]ui.DeviceInfo, len(devices))
	for i, d := range devices {
		uiDevices[i] = ui.DeviceInfo{
			VendorID:     d.VendorID,
			ProductID:    d.ProductID,
			Manufacturer: d.Manufacturer,
			Product:      d.Product,
		}
	}
	ui.PrintDeviceList(uiDevices)
}

// runSetDevice handles the set-device subcommand
func runSetDevice(args []string) {
	fs := flag.NewFlagSet("set-device", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to configuration file")
	fs.Usage = func() {
		ui.PrintSetDeviceUsage()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	remaining := fs.Args()

	var vendorID, productID uint16

	if len(remaining) >= 2 {
		// Parse provided IDs
		vid, err := parseID(remaining[0])
		if err != nil {
			ui.PrintFatalError("Invalid vendor_id", fmt.Sprintf("%q: %v", remaining[0], err))
			os.Exit(1)
		}
		pid, err := parseID(remaining[1])
		if err != nil {
			ui.PrintFatalError("Invalid product_id", fmt.Sprintf("%q: %v", remaining[1], err))
			os.Exit(1)
		}
		vendorID = vid
		productID = pid
	} else if len(remaining) == 1 {
		ui.PrintFatalError("Invalid arguments", "Both vendor_id and product_id must be provided, or neither")
		os.Exit(1)
	} else {
		// Interactive selection
		device, err := selectDevice()
		if err != nil {
			ui.PrintFatalError("Device selection failed", err.Error())
			os.Exit(1)
		}
		if device == nil {
			fmt.Println(ui.Muted("No device selected"))
			os.Exit(0)
		}
		vendorID = device.VendorID
		productID = device.ProductID
	}

	// Update or create config file
	if config.Exists(*configPath) {
		if err := config.UpdateDeviceIDs(*configPath, vendorID, productID); err != nil {
			ui.PrintFatalError("Failed to update config", err.Error())
			os.Exit(1)
		}
		ui.PrintDeviceUpdated(*configPath, vendorID, productID)
	} else {
		if err := config.CreateDefaultConfig(*configPath, vendorID, productID); err != nil {
			ui.PrintFatalError("Failed to create config", err.Error())
			os.Exit(1)
		}
		ui.PrintDeviceCreated(*configPath, vendorID, productID)
	}
}

// runConfigPush handles the config-push subcommand
func runConfigPush(args []string) {
	fs := flag.NewFlagSet("config-push", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to configuration file")
	fs.Usage = func() {
		ui.PrintConfigPushUsage()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	ui.PrintConfigPushProgress(fmt.Sprintf("Reading config from %s...", *configPath))

	cfg, err := config.Load(*configPath)
	if err != nil {
		ui.PrintFatalError("Failed to load config", err.Error())
		os.Exit(1)
	}

	ui.PrintConfigPushProgress(fmt.Sprintf("Converting %d button mapping(s)...", len(cfg.Buttons)))

	mountPoint, err := configpush.FindCIRCUITPY()
	if err != nil {
		ui.PrintFatalError("Failed to find device", err.Error())
		os.Exit(1)
	}

	ui.PrintConfigPushProgress(fmt.Sprintf("Found CIRCUITPY at %s", mountPoint))

	if err := configpush.Push(cfg); err != nil {
		ui.PrintFatalError("Failed to push config", err.Error())
		os.Exit(1)
	}

	ui.PrintConfigPushSuccess(mountPoint, len(cfg.Buttons))
}

// parseID parses a vendor or product ID from string (supports hex with 0x prefix or decimal)
func parseID(s string) (uint16, error) {
	s = strings.TrimSpace(s)

	var val uint64
	var err error

	if strings.HasPrefix(strings.ToLower(s), "0x") {
		val, err = strconv.ParseUint(s[2:], 16, 16)
	} else {
		val, err = strconv.ParseUint(s, 10, 16)
	}

	if err != nil {
		return 0, err
	}

	return uint16(val), nil
}

// selectDevice displays an interactive device selection menu using huh
func selectDevice() (*ui.DeviceInfo, error) {
	devices, err := hid.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no HID devices found")
	}

	// Deduplicate devices by vendor/product ID
	seen := make(map[uint32]bool)
	var unique []ui.DeviceInfo

	for _, d := range devices {
		key := uint32(d.VendorID)<<16 | uint32(d.ProductID)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Skip devices with no vendor/product ID
		if d.VendorID == 0 && d.ProductID == 0 {
			continue
		}

		unique = append(unique, ui.DeviceInfo{
			VendorID:     d.VendorID,
			ProductID:    d.ProductID,
			Manufacturer: d.Manufacturer,
			Product:      d.Product,
		})
	}

	if len(unique) == 0 {
		return nil, fmt.Errorf("no identifiable HID devices found")
	}

	return ui.SelectDevice(unique)
}

type App struct {
	config         *config.Config
	verbose        bool
	hidDevice      *hid.Device
	gestureEngine  *gesture.Engine
	actionMapper   *action.Mapper
	actionExecutor *action.Executor
	ptyManager     *pty.Manager
	displayManager *display.Manager
}

func newApp(cfg *config.Config, verbose bool) (*App, error) {
	app := &App{
		config:  cfg,
		verbose: verbose,
	}

	// Initialize HID device
	hidDevice, err := hid.NewDevice(cfg.Device.VendorID, cfg.Device.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to open HID device: %w", err)
	}
	app.hidDevice = hidDevice

	// Initialize action mapper
	app.actionMapper = action.NewMapper(cfg)

	// Initialize PTY manager
	ptyManager, err := pty.NewManager(cfg.TUI.Command, cfg.TUI.Args, cfg.TUI.WorkingDir)
	if err != nil {
		hidDevice.Close()
		return nil, fmt.Errorf("failed to create PTY manager: %w", err)
	}
	app.ptyManager = ptyManager

	// Initialize action executor
	app.actionExecutor = action.NewExecutor(ptyManager)

	// Initialize gesture engine
	app.gestureEngine = gesture.NewEngine(cfg.Timing, func(g gesture.Gesture) {
		if verbose {
			log.Printf("Gesture detected: %s", g)
		}
		keys := app.actionMapper.Map(g)
		if len(keys) > 0 {
			if err := app.actionExecutor.Execute(keys); err != nil {
				log.Printf("Failed to execute action: %v", err)
			}
		}
	})

	// Initialize display manager
	app.displayManager = display.NewManager(cfg.Display, hidDevice)

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	// Start PTY
	if err := a.ptyManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}

	// Start display manager
	a.displayManager.Start(ctx, a.ptyManager)

	// Start gesture engine
	a.gestureEngine.Start(ctx)

	// Start reading from HID device
	events := make(chan hid.Event, 64)
	go func() {
		if err := a.hidDevice.ReadEvents(ctx, events); err != nil && ctx.Err() == nil {
			log.Printf("HID read error: %v", err)
		}
		close(events)
	}()

	// Process HID events
	for {
		select {
		case <-ctx.Done():
			a.shutdown()
			return nil
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("HID device disconnected")
			}
			a.gestureEngine.ProcessEvent(event)
		}
	}
}

func (a *App) shutdown() {
	if a.verbose {
		log.Println("Shutting down...")
	}
	a.gestureEngine.Stop()
	a.displayManager.Stop()
	a.ptyManager.Stop()
	a.hidDevice.Close()
}
