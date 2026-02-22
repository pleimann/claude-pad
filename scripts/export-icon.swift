#!/usr/bin/env swift
/// Converts assets/CamelPad.icon (Icon Composer bundle) → assets/CamelPad.icns
/// Reads icon.json for fill color and finds the source PNG in Assets/.
/// Usage: swift scripts/export-icon.swift [--force]
import AppKit
import Foundation

let args = CommandLine.arguments
let force = args.contains("--force")

let iconBundleURL = URL(fileURLWithPath: "assets/CamelPad.icon")
let outputURL = URL(fileURLWithPath: "assets/CamelPad.icns")

// Skip if output is newer than source (unless --force)
if !force,
   let outMod = try? outputURL.resourceValues(forKeys: [.contentModificationDateKey]).contentModificationDate,
   let srcMod = try? iconBundleURL.resourceValues(forKeys: [.contentModificationDateKey]).contentModificationDate,
   outMod >= srcMod {
    print("assets/CamelPad.icns is up to date.")
    exit(0)
}

// --- Parse icon.json ---

struct IconJSON: Decodable {
    struct Fill: Decodable {
        let automaticGradient: String?
        enum CodingKeys: String, CodingKey { case automaticGradient = "automatic-gradient" }
    }
    struct Layer: Decodable {
        let imageName: String
        enum CodingKeys: String, CodingKey { case imageName = "image-name" }
    }
    struct Group: Decodable {
        let layers: [Layer]
    }
    let fill: Fill?
    let groups: [Group]
}

let jsonURL = iconBundleURL.appendingPathComponent("icon.json")
guard let jsonData = try? Data(contentsOf: jsonURL),
      let iconJSON = try? JSONDecoder().decode(IconJSON.self, from: jsonData) else {
    fputs("Error: Could not read or parse \(jsonURL.path)\n", stderr)
    exit(1)
}

// Find source PNG name from first layer
guard let imageName = iconJSON.groups.first?.layers.first?.imageName else {
    fputs("Error: No image layer found in icon.json\n", stderr)
    exit(1)
}

let pngURL = iconBundleURL.appendingPathComponent("Assets").appendingPathComponent(imageName)
guard let sourceImage = NSImage(contentsOf: pngURL) else {
    fputs("Error: Could not load \(pngURL.path)\n", stderr)
    exit(1)
}

// Parse fill color from "extended-srgb:R,G,B,A"
var fillColor = NSColor(red: 0, green: 0.533, blue: 1.0, alpha: 1.0)
if let gradientStr = iconJSON.fill?.automaticGradient,
   gradientStr.hasPrefix("extended-srgb:") {
    let components = gradientStr.dropFirst("extended-srgb:".count)
        .split(separator: ",").compactMap { Double($0) }
    if components.count == 4 {
        fillColor = NSColor(colorSpace: .extendedSRGB,
                           components: components.map { CGFloat($0) }, count: 4)
            ?? fillColor
    }
}

// Derive gradient: top is lightened, bottom is the fill color
let topColor = fillColor.blended(withFraction: 0.3, of: .white) ?? fillColor

// --- Render icon at a given pixel size ---

func renderIcon(size: Int) -> Data {
    let s = CGFloat(size)
    let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil,
        pixelsWide: size, pixelsHigh: size,
        bitsPerSample: 8, samplesPerPixel: 4,
        hasAlpha: true, isPlanar: false,
        colorSpaceName: .deviceRGB,
        bytesPerRow: 0, bitsPerPixel: 0
    )!
    rep.size = NSSize(width: s, height: s)

    NSGraphicsContext.saveGraphicsState()
    let ctx = NSGraphicsContext(bitmapImageRep: rep)!
    NSGraphicsContext.current = ctx
    let cgCtx = ctx.cgContext

    let rect = CGRect(x: 0, y: 0, width: s, height: s)
    // macOS Big Sur+ rounded rect: ~22.5% corner radius
    let radius = s * 0.225
    let path = CGPath(roundedRect: rect, cornerWidth: radius, cornerHeight: radius, transform: nil)

    // Clip to rounded rect
    cgCtx.addPath(path)
    cgCtx.clip()

    // Draw vertical gradient background (top-light → bottom-fill)
    let gradient = CGGradient(
        colorsSpace: CGColorSpace(name: CGColorSpace.extendedSRGB)!,
        colors: [topColor.cgColor, fillColor.cgColor] as CFArray,
        locations: [0.0, 1.0]
    )!
    cgCtx.drawLinearGradient(gradient,
                             start: CGPoint(x: s / 2, y: s),
                             end: CGPoint(x: s / 2, y: 0),
                             options: [])

    // Draw source image centered with padding (~10%)
    let padding = s * 0.10
    let imgRect = CGRect(x: padding, y: padding, width: s - padding * 2, height: s - padding * 2)
    sourceImage.draw(in: imgRect, from: .zero, operation: .sourceOver, fraction: 1.0)

    NSGraphicsContext.restoreGraphicsState()

    return rep.representation(using: .png, properties: [:])!
}

// --- Build iconset and convert ---

// iconutil requires these exact filenames
let sizes: [(Int, String)] = [
    (16,   "icon_16x16.png"),
    (32,   "icon_16x16@2x.png"),
    (32,   "icon_32x32.png"),
    (64,   "icon_32x32@2x.png"),
    (128,  "icon_128x128.png"),
    (256,  "icon_128x128@2x.png"),
    (256,  "icon_256x256.png"),
    (512,  "icon_256x256@2x.png"),
    (512,  "icon_512x512.png"),
    (1024, "icon_512x512@2x.png"),
]

let tmpIconset = URL(fileURLWithPath: NSTemporaryDirectory())
    .appendingPathComponent("CamelPad_\(Int.random(in: 100000...999999)).iconset")

do {
    try FileManager.default.createDirectory(at: tmpIconset, withIntermediateDirectories: true)
} catch {
    fputs("Error: Could not create temp directory: \(error)\n", stderr)
    exit(1)
}

defer { try? FileManager.default.removeItem(at: tmpIconset) }

for (size, filename) in sizes {
    let png = renderIcon(size: size)
    do {
        try png.write(to: tmpIconset.appendingPathComponent(filename))
    } catch {
        fputs("Error: Could not write \(filename): \(error)\n", stderr)
        exit(1)
    }
}

let iconutil = Process()
iconutil.executableURL = URL(fileURLWithPath: "/usr/bin/iconutil")
iconutil.arguments = ["-c", "icns", tmpIconset.path, "-o", outputURL.path]
do {
    try iconutil.run()
    iconutil.waitUntilExit()
} catch {
    fputs("Error: Could not run iconutil: \(error)\n", stderr)
    exit(1)
}

guard iconutil.terminationStatus == 0 else {
    fputs("Error: iconutil exited with status \(iconutil.terminationStatus)\n", stderr)
    exit(1)
}

print("Created \(outputURL.path)")
