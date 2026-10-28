import AppKit

let destination = CommandLine.arguments[1]
guard CommandLine.arguments.count == 3,
      let source = NSImage(contentsOfFile: CommandLine.arguments[2]) else {
    fatalError("Expected iconset destination and source PNG")
}
let sizes = [(16, "16x16"), (32, "16x16@2x"), (32, "32x32"), (64, "32x32@2x"), (128, "128x128"), (256, "128x128@2x"), (256, "256x256"), (512, "256x256@2x"), (512, "512x512"), (1024, "512x512@2x")]

for (size, filename) in sizes {
    let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: size, pixelsHigh: size, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
    let context = NSGraphicsContext(bitmapImageRep: bitmap)!
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    context.cgContext.scaleBy(x: CGFloat(size) / 1024, y: CGFloat(size) / 1024)
    // Standard macOS padding and rounded mask; preserve the source artwork.
    let bounds = NSRect(x: 64, y: 64, width: 896, height: 896)
    context.cgContext.clear(CGRect(x: 0, y: 0, width: 1024, height: 1024))
    context.imageInterpolation = .high
    context.cgContext.setShouldAntialias(true)
    NSBezierPath(roundedRect: bounds, xRadius: 192, yRadius: 192).addClip()
    source.draw(in: bounds, from: .zero, operation: .copy, fraction: 1)
    NSGraphicsContext.restoreGraphicsState()
    try bitmap.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: destination).appendingPathComponent("icon_\(filename).png"))
}
