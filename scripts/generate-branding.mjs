import { deflateSync } from "node:zlib"
import { mkdir, writeFile } from "node:fs/promises"

// Dependency-free rasterization of branding/logo.svg's simple geometry.
const crc = (bytes) => {
  let value = 0xffffffff
  for (const byte of bytes) {
    value ^= byte
    for (let bit = 0; bit < 8; bit++) value = (value >>> 1) ^ (value & 1 ? 0xedb88320 : 0)
  }
  return (value ^ 0xffffffff) >>> 0
}
const chunk = (name, data) => {
  const type = Buffer.from(name)
  const result = Buffer.alloc(data.length + 12)
  result.writeUInt32BE(data.length)
  type.copy(result, 4)
  data.copy(result, 8)
  result.writeUInt32BE(crc(Buffer.concat([type, data])), data.length + 8)
  return result
}
const inside = (x, y, points) => {
  let result = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i++) {
    const [xi, yi] = points[i],
      [xj, yj] = points[j]
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) result = !result
  }
  return result
}
const png = (size) => {
  const rows = Buffer.alloc((size * 4 + 1) * size)
  const glyph = [
    [67, 54],
    [201, 54],
    [184, 89],
    [109, 89],
    [109, 120],
    [179, 120],
    [163, 154],
    [109, 154],
    [109, 202],
    [67, 202],
  ]
  for (let y = 0; y < size; y++)
    for (let x = 0; x < size; x++) {
      const color = [0, 0, 0, 0]
      for (let sy = 0; sy < 4; sy++)
        for (let sx = 0; sx < 4; sx++) {
          const px = ((x + (sx + 0.5) / 4) * 256) / size,
            py = ((y + (sy + 0.5) / 4) * 256) / size
          const dx = Math.max(54 - px, 0, px - 202),
            dy = Math.max(54 - py, 0, py - 202)
          if (dx * dx + dy * dy > 54 * 54) continue
          let sample = [17, 24, 23, 255]
          if (inside(px, py, glyph)) {
            const t = (px + py) / 512
            sample = [110 - 90 * t, 231 - 47 * t, 183 - 17 * t, 255]
          }
          if (
            inside(px, py, [
              [184, 54],
              [201, 54],
              [184, 89],
              [167, 89],
            ])
          )
            sample = [167, 243, 208, 255]
          for (let i = 0; i < 4; i++) color[i] += sample[i] / 16
        }
      const offset = y * (size * 4 + 1) + 1 + x * 4
      for (let i = 0; i < 4; i++) rows[offset + i] = Math.round(color[i])
    }
  const header = Buffer.alloc(13)
  header.writeUInt32BE(size)
  header.writeUInt32BE(size, 4)
  header[8] = 8
  header[9] = 6
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(rows)),
    chunk("IEND", Buffer.alloc(0)),
  ])
}
await mkdir("build/windows", { recursive: true })
await mkdir("frontend/public", { recursive: true })
const sizes = [16, 32, 48, 64, 128, 256],
  images = sizes.map(png)
const header = Buffer.alloc(6 + sizes.length * 16)
header.writeUInt16LE(1, 2)
header.writeUInt16LE(sizes.length, 4)
let offset = header.length
sizes.forEach((size, i) => {
  const p = 6 + i * 16
  header[p] = size % 256
  header[p + 1] = size % 256
  header.writeUInt16LE(1, p + 4)
  header.writeUInt16LE(32, p + 6)
  header.writeUInt32LE(images[i].length, p + 8)
  header.writeUInt32LE(offset, p + 12)
  offset += images[i].length
})
await writeFile("branding/logo.png", png(256))
await writeFile("build/windows/icon.ico", Buffer.concat([header, ...images]))
await writeFile("frontend/public/favicon.svg", await (await import("node:fs/promises")).readFile("branding/logo.svg"))
