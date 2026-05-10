import AppKit
import Foundation

let args = Array(CommandLine.arguments.dropFirst())
guard args.count >= 2 else {
    throw NSError(
        domain: "EasyLLMIcon",
        code: 2,
        userInfo: [NSLocalizedDescriptionKey: "usage: make-app-icon <source.png> <iconset-dir> [output.icns] [canonical-1024.png] [crop-inset-ratio]"]
    )
}

let sourceURL = URL(fileURLWithPath: args[0])
let outputURL = URL(fileURLWithPath: args[1], isDirectory: true)
let icnsURL = args.count >= 3 ? URL(fileURLWithPath: args[2]) : nil
let canonicalURL = args.count >= 4 ? URL(fileURLWithPath: args[3]) : nil
let cropInsetRatio = args.count >= 5 ? max(0, min(0.2, Double(args[4]) ?? 0)) : 0

guard let sourceImage = NSImage(contentsOf: sourceURL) else {
    throw NSError(
        domain: "EasyLLMIcon",
        code: 3,
        userInfo: [NSLocalizedDescriptionKey: "failed to read source icon: \(sourceURL.path)"]
    )
}

try FileManager.default.createDirectory(at: outputURL, withIntermediateDirectories: true)

let iconFiles: [(name: String, pixels: Int)] = [
    ("icon_16x16.png", 16),
    ("icon_16x16@2x.png", 32),
    ("icon_32x32.png", 32),
    ("icon_32x32@2x.png", 64),
    ("icon_128x128.png", 128),
    ("icon_128x128@2x.png", 256),
    ("icon_256x256.png", 256),
    ("icon_256x256@2x.png", 512),
    ("icon_512x512.png", 512),
    ("icon_512x512@2x.png", 1024),
]

var icnsEntries: [(type: String, data: Data)] = []
let icnsTypesByFile = [
    "icon_16x16.png": "icp4",
    "icon_32x32.png": "icp5",
    "icon_32x32@2x.png": "icp6",
    "icon_128x128.png": "ic07",
    "icon_256x256.png": "ic08",
    "icon_512x512.png": "ic09",
    "icon_512x512@2x.png": "ic10",
]

if let canonicalURL {
    let canonicalImage = renderIcon(from: sourceImage, pixels: 1024, cropInsetRatio: cropInsetRatio)
    try pngData(from: canonicalImage).write(to: canonicalURL)
}

for icon in iconFiles {
    let image = renderIcon(from: sourceImage, pixels: icon.pixels, cropInsetRatio: cropInsetRatio)
    let targetURL = outputURL.appendingPathComponent(icon.name)
    let png = try pngData(from: image)
    try png.write(to: targetURL)
    if let icnsType = icnsTypesByFile[icon.name] {
        icnsEntries.append((type: icnsType, data: png))
    }
}

if let icnsURL {
    try writeICNS(entries: icnsEntries, to: icnsURL)
}

private func renderIcon(from source: NSImage, pixels: Int, cropInsetRatio: Double) -> NSImage {
    let size = CGFloat(pixels)
    let image = NSImage(size: NSSize(width: size, height: size))
    image.lockFocus()
    NSColor.clear.setFill()
    NSRect(x: 0, y: 0, width: size, height: size).fill()

    let rect = NSRect(x: 0, y: 0, width: size, height: size)
    let radius = size * 0.22
    let clip = NSBezierPath(roundedRect: rect, xRadius: radius, yRadius: radius)
    clip.addClip()

    let sourceSize = source.size
    let shortest = min(sourceSize.width, sourceSize.height)
    let cropInset = shortest * CGFloat(cropInsetRatio)
    let cropSize = max(1, shortest - cropInset * 2)
    let sourceRect = NSRect(
        x: (sourceSize.width - cropSize) / 2,
        y: (sourceSize.height - cropSize) / 2,
        width: cropSize,
        height: cropSize
    )

    source.draw(
        in: rect,
        from: sourceRect,
        operation: .sourceOver,
        fraction: 1,
        respectFlipped: true,
        hints: [.interpolation: NSImageInterpolation.high]
    )

    image.unlockFocus()
    return image
}

private func pngData(from image: NSImage) throws -> Data {
    guard let tiff = image.tiffRepresentation,
          let bitmap = NSBitmapImageRep(data: tiff),
          let data = bitmap.representation(using: .png, properties: [:]) else {
        throw NSError(
            domain: "EasyLLMIcon",
            code: 1,
            userInfo: [NSLocalizedDescriptionKey: "failed to render PNG"]
        )
    }
    return data
}

private func writeICNS(entries: [(type: String, data: Data)], to url: URL) throws {
    var result = Data()
    let totalLength = 8 + entries.reduce(0) { $0 + 8 + $1.data.count }
    appendOSType("icns", to: &result)
    appendUInt32BE(UInt32(totalLength), to: &result)

    for entry in entries {
        appendOSType(entry.type, to: &result)
        appendUInt32BE(UInt32(entry.data.count + 8), to: &result)
        result.append(entry.data)
    }

    try result.write(to: url)
}

private func appendOSType(_ value: String, to data: inout Data) {
    let bytes = Array(value.utf8.prefix(4))
    data.append(contentsOf: bytes)
    if bytes.count < 4 {
        data.append(contentsOf: Array(repeating: 0, count: 4 - bytes.count))
    }
}

private func appendUInt32BE(_ value: UInt32, to data: inout Data) {
    data.append(UInt8((value >> 24) & 0xff))
    data.append(UInt8((value >> 16) & 0xff))
    data.append(UInt8((value >> 8) & 0xff))
    data.append(UInt8(value & 0xff))
}
