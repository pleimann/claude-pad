package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// DeviceInfo contains information about a HID device for display
type DeviceInfo struct {
	VendorID     uint16
	ProductID    uint16
	Manufacturer string
	Product      string
}

// deviceSelectModel wraps huh form in Bubble Tea for proper escape handling
type deviceSelectModel struct {
	form     *huh.Form
	devices  []DeviceInfo
	selected int
	aborted  bool
}

func (m deviceSelectModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m deviceSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.aborted = true
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		return m, tea.Quit
	}

	return m, cmd
}

func (m deviceSelectModel) View() string {
	if m.form.State == huh.StateCompleted {
		return ""
	}
	return m.form.View()
}

// SelectDevice presents an interactive device selection using huh with Bubble Tea
func SelectDevice(devices []DeviceInfo) (*DeviceInfo, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices to select from")
	}

	options := make([]huh.Option[int], len(devices))
	for i, d := range devices {
		name := formatDeviceName(d)
		label := fmt.Sprintf("%s  %s",
			DeviceIDStyle.Render(fmt.Sprintf("0x%04X:0x%04X", d.VendorID, d.ProductID)),
			name,
		)
		options[i] = huh.NewOption(label, i)
	}

	var selectedIndex int

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select HID Device").
				Description("Choose the macropad device to configure (esc to cancel)").
				Options(options...).
				Value(&selectedIndex),
		),
	).WithTheme(customTheme()).WithShowHelp(false)

	model := deviceSelectModel{
		form:    form,
		devices: devices,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(deviceSelectModel)
	if m.aborted {
		return nil, nil // User cancelled
	}

	return &devices[selectedIndex], nil
}

// formatDeviceName creates a readable name for the device
func formatDeviceName(d DeviceInfo) string {
	name := d.Product
	if name == "" {
		name = "Unknown Device"
	}
	if d.Manufacturer != "" {
		name = d.Manufacturer + " " + name
	}
	return name
}

// PrintDeviceList displays a styled list of HID devices
func PrintDeviceList(devices []DeviceInfo) {
	if len(devices) == 0 {
		fmt.Println(Warning("No HID devices found"))
		return
	}

	// Header
	fmt.Println()
	fmt.Println(Title("HID Devices"))
	fmt.Println(Muted(fmt.Sprintf("Found %d device(s)", len(devices))))
	fmt.Println()

	// Device list
	for _, d := range devices {
		printDevice(d)
	}
	fmt.Println()
}

func printDevice(d DeviceInfo) {
	// ID line
	idLine := DeviceIDStyle.Render(fmt.Sprintf("  0x%04X:0x%04X", d.VendorID, d.ProductID))

	// Name and manufacturer
	name := d.Product
	if name == "" {
		name = "Unknown Device"
	}

	var details []string
	details = append(details, DeviceNameStyle.Render(name))
	if d.Manufacturer != "" {
		details = append(details, DeviceManufacturerStyle.Render("by "+d.Manufacturer))
	}

	fmt.Printf("%s  %s\n", idLine, strings.Join(details, " "))
}

// PrintDeviceUpdated shows a success message after updating device config
func PrintDeviceUpdated(configPath string, vendorID, productID uint16) {
	fmt.Println()
	fmt.Println(Success("Device configuration updated"))
	fmt.Println()
	fmt.Printf("  %s %s\n", Muted("Config:"), configPath)
	fmt.Printf("  %s %s\n", Muted("Device:"), DeviceIDStyle.Render(fmt.Sprintf("0x%04X:0x%04X", vendorID, productID)))
	fmt.Println()
}

// PrintDeviceCreated shows a success message after creating device config
func PrintDeviceCreated(configPath string, vendorID, productID uint16) {
	fmt.Println()
	fmt.Println(Success("Device configuration created"))
	fmt.Println()
	fmt.Printf("  %s %s\n", Muted("Config:"), configPath)
	fmt.Printf("  %s %s\n", Muted("Device:"), DeviceIDStyle.Render(fmt.Sprintf("0x%04X:0x%04X", vendorID, productID)))
	fmt.Println()
}

// customTheme returns a custom huh theme matching our style palette
func customTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Customize the theme to match our color palette
	t.Focused.Title = t.Focused.Title.Foreground(ColorPrimary).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(ColorMuted)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ColorPrimary)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(lipgloss.Color("#F9FAFB"))
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ColorPrimary)

	return t
}
