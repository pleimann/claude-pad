package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pleimann/camel-pad/internal/utils"
)

// PrintUsage displays the styled help/usage text
func PrintUsage(version string) {
	// Title banner
	banner := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(utils.ExecutableName())

	versionTag := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("v" + version)

	fmt.Printf("%s %s\n", banner, versionTag)
	fmt.Println(Muted("Macropad middleware for TUI applications"))
	fmt.Println()

	// Usage section
	printSection("Usage", []string{
		utils.ExecutableName() + " [flags]              Run the middleware",
		utils.ExecutableName() + " list-devices         List available HID devices",
		utils.ExecutableName() + " set-device [args]    Configure the HID device",
		utils.ExecutableName() + " config-push          Push config to CircuitPython device",
		utils.ExecutableName() + " help                 Show this help message",
	})

	// Flags section
	printSection("Flags", []string{
		"-config string    Path to configuration file (default \"config.yaml\")",
		"-verbose          Enable verbose logging",
		"-version          Print version and exit",
	})

	// Commands section
	printCommandSection()

	// Examples section
	printExamplesSection()
}

func printSection(title string, items []string) {
	fmt.Println(Bold(title))
	for _, item := range items {
		fmt.Printf("  %s\n", item)
	}
	fmt.Println()
}

func printCommandSection() {
	fmt.Println(Bold("Commands"))

	cmdStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	fmt.Printf("  %s\n", cmdStyle.Render("list-devices"))
	fmt.Printf("      List available HID devices\n")
	fmt.Println()

	fmt.Printf("  %s\n", cmdStyle.Render("set-device"))
	fmt.Printf("      Set the HID device in the config file\n")
	fmt.Printf("      Run %s for more information\n", Code(utils.ExecutableName()+" set-device --help"))
	fmt.Println()

	fmt.Printf("  %s\n", cmdStyle.Render("config-push"))
	fmt.Printf("      Push button configuration to the CircuitPython device\n")
	fmt.Printf("      Run %s for more information\n", Code(utils.ExecutableName()+" config-push --help"))
	fmt.Println()
}

func printExamplesSection() {
	fmt.Println(Bold("Examples"))

	examples := []struct {
		cmd  string
		desc string
	}{
		{utils.ExecutableName(), "Run with default config.yaml"},
		{utils.ExecutableName() + " -config my.yaml", "Run with custom config file"},
		{utils.ExecutableName() + " list-devices", "List connected HID devices"},
		{utils.ExecutableName() + " set-device", "Interactive device selection"},
		{utils.ExecutableName() + " set-device 0x1234 0x5678", "Set device by vendor/product ID"},
	}

	cmdStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary)

	maxLen := 0
	for _, ex := range examples {
		if len(ex.cmd) > maxLen {
			maxLen = len(ex.cmd)
		}
	}

	for _, ex := range examples {
		padding := strings.Repeat(" ", maxLen-len(ex.cmd)+2)
		fmt.Printf("  %s%s%s\n", cmdStyle.Render(ex.cmd), padding, Muted(ex.desc))
	}
	fmt.Println()
}

// PrintSetDeviceUsage displays the styled help text for set-device subcommand
func PrintSetDeviceUsage() {
	fmt.Println(Bold("Usage:"), utils.ExecutableName()+" set-device [options] [vendor_id product_id]")
	fmt.Println()
	fmt.Println("Set the HID device in the configuration file.")
	fmt.Println()
	fmt.Println(Muted("If vendor_id and product_id are provided, updates the config directly."))
	fmt.Println(Muted("Otherwise, displays a list of connected devices to choose from."))
	fmt.Println()

	fmt.Println(Bold("Arguments"))
	fmt.Printf("  %s    Device vendor ID (hex with 0x prefix or decimal)\n", SubtitleStyle.Render("vendor_id"))
	fmt.Printf("  %s   Device product ID (hex with 0x prefix or decimal)\n", SubtitleStyle.Render("product_id"))
	fmt.Println()

	fmt.Println(Bold("Options"))
	fmt.Printf("  %s    Path to configuration file (default \"config.yaml\")\n", SubtitleStyle.Render("-config string"))
	fmt.Println()

	fmt.Println(Bold("Examples"))
	examples := []struct {
		cmd  string
		desc string
	}{
		{utils.ExecutableName() + " set-device", "Interactive selection"},
		{utils.ExecutableName() + " set-device 0x1234 0x5678", "Direct specification"},
		{utils.ExecutableName() + " set-device -config my.yaml", "Use different config"},
	}

	cmdStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary)

	maxLen := 0
	for _, ex := range examples {
		if len(ex.cmd) > maxLen {
			maxLen = len(ex.cmd)
		}
	}

	for _, ex := range examples {
		padding := strings.Repeat(" ", maxLen-len(ex.cmd)+2)
		fmt.Printf("  %s%s%s\n", cmdStyle.Render(ex.cmd), padding, Muted(ex.desc))
	}
	fmt.Println()
}

// PrintVersion displays the styled version information
func PrintVersion(version string) {
	banner := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(utils.ExecutableName())

	versionTag := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Render("v" + version)

	fmt.Printf("%s %s\n", banner, versionTag)
}

// PrintError displays a styled error message
func PrintError(message string) {
	fmt.Println(Error(message))
}

// PrintFatalError displays a styled fatal error message with context
func PrintFatalError(context, message string) {
	fmt.Println()
	fmt.Println(Error(context))
	fmt.Printf("  %s\n", Muted(message))
	fmt.Println()
}

// PrintConfigPushUsage displays the styled help text for config-push subcommand
func PrintConfigPushUsage() {
	fmt.Println(Bold("Usage:"), utils.ExecutableName()+" config-push [options]")
	fmt.Println()
	fmt.Println("Push button configuration to the CircuitPython device.")
	fmt.Println()
	fmt.Println(Muted("Converts button mappings from config.yaml to Python format"))
	fmt.Println(Muted("and writes config.py to the CIRCUITPY drive."))
	fmt.Println()

	fmt.Println(Bold("Options"))
	fmt.Printf("  %s    Path to configuration file (default \"config.yaml\")\n", SubtitleStyle.Render("-config string"))
	fmt.Println()

	fmt.Println(Bold("Examples"))
	examples := []struct {
		cmd  string
		desc string
	}{
		{utils.ExecutableName() + " config-push", "Push using default config.yaml"},
		{utils.ExecutableName() + " config-push -config my.yaml", "Push using custom config"},
	}

	cmdStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary)

	maxLen := 0
	for _, ex := range examples {
		if len(ex.cmd) > maxLen {
			maxLen = len(ex.cmd)
		}
	}

	for _, ex := range examples {
		padding := strings.Repeat(" ", maxLen-len(ex.cmd)+2)
		fmt.Printf("  %s%s%s\n", cmdStyle.Render(ex.cmd), padding, Muted(ex.desc))
	}
	fmt.Println()
}

// PrintConfigPushProgress prints progress during config push
func PrintConfigPushProgress(message string) {
	fmt.Printf("  %s %s\n", Muted("→"), message)
}

// PrintConfigPushSuccess prints the success message after config push
func PrintConfigPushSuccess(mountPoint string, buttonCount int) {
	fmt.Println()
	fmt.Println(Success("Configuration pushed successfully"))
	fmt.Printf("  %s %s\n", Muted("Location:"), mountPoint+"/config.py")
	fmt.Printf("  %s %d button(s) configured\n", Muted("Buttons:"), buttonCount)
	fmt.Println()
	fmt.Println(Muted("Restart your device to apply changes."))
}
