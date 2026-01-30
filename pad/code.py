import board
import displayio
import vectorio

# Display is already initialized - just use it
display = board.DISPLAY
print(f"Display: {display.width}x{display.height}")

# Simple test - fill with colors

# Create a simple colored rectangle
palette = displayio.Palette(1)
palette[0] = 0xFF0000  # Red

rect = vectorio.Rectangle(
    pixel_shader=palette,
    width=display.width,
    height=display.height // 3,
    x=0,
    y=0
)

palette2 = displayio.Palette(1)
palette2[0] = 0x00FF00  # Green

rect2 = vectorio.Rectangle(
    pixel_shader=palette2,
    width=display.width,
    height=display.height // 3,
    x=0,
    y=display.height // 3
)

palette3 = displayio.Palette(1)
palette3[0] = 0x0000FF  # Blue

rect3 = vectorio.Rectangle(
    pixel_shader=palette3,
    width=display.width,
    height=display.height // 3,
    x=0,
    y=2 * display.height // 3
)

# Create a group and add shapes
group = displayio.Group()
group.append(rect)
group.append(rect2)
group.append(rect3)

display.root_group = group

print("Display test complete - showing RGB stripes")