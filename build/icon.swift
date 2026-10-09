import AppKit

let destination = CommandLine.arguments[1]
let sizes = [(16, "16x16"), (32, "16x16@2x"), (32, "32x32"), (64, "32x32@2x"), (128, "128x128"), (256, "128x128@2x"), (256, "256x256"), (512, "256x256@2x"), (512, "512x512"), (1024, "512x512@2x")]

for (size, filename) in sizes {
    let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: size, pixelsHigh: size, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
    let context = NSGraphicsContext(bitmapImageRep: bitmap)!
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    context.cgContext.scaleBy(x: CGFloat(size) / 1024, y: CGFloat(size) / 1024)
    NSColor(calibratedRed: 0.21, green: 0.44, blue: 0.86, alpha: 1).setFill()
    NSBezierPath(roundedRect: NSRect(x: 64, y: 64, width: 896, height: 896), xRadius: 192, yRadius: 192).fill()
    NSColor.white.setFill()
    for x in [242.0, 522.0] {
        NSBezierPath(roundedRect: NSRect(x: x, y: 280, width: 260, height: 444), xRadius: 34, yRadius: 34).fill()
    }
    NSColor(calibratedRed: 0.75, green: 0.84, blue: 1, alpha: 1).setFill()
    for x in [286.0, 566.0] {
        for y in [426.0, 500.0, 574.0] {
            NSBezierPath(roundedRect: NSRect(x: x, y: y, width: 172, height: 16), xRadius: 8, yRadius: 8).fill()
        }
    }
    NSGraphicsContext.restoreGraphicsState()
    try bitmap.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: destination).appendingPathComponent("icon_\(filename).png"))
}
