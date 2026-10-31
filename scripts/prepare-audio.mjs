import { mkdir, readFile, writeFile } from "node:fs/promises"
import { gunzipSync } from "node:zlib"
import { createHash } from "node:crypto"

const release = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1"
const root = "internal/bundle/files"
await mkdir(root, { recursive: true })
const assets = [
  ["ffmpeg-win32-x64.gz", "ffmpeg.exe"],
  ["win32-x64.LICENSE", "ffmpeg-LICENSE.txt"],
  ["win32-x64.README", "ffmpeg-README.txt"],
]
const checksums = JSON.parse(await readFile(new URL("./audio-checksums.json", import.meta.url), "utf8"))
for (const [asset, name] of assets) {
  let bytes
  try {
    bytes = await readFile(`${root}/${name}`)
  } catch {
    /* download below */
  }
  const digest = (value) => createHash("sha256").update(value).digest("hex")
  if (bytes && digest(bytes) === checksums[name]) continue
  const response = await fetch(`${release}/${asset}`, { signal: AbortSignal.timeout(120000) })
  if (!response.ok) throw new Error(`Cannot download ${asset}: HTTP ${response.status}`)
  bytes = Buffer.from(await response.arrayBuffer())
  if (asset.endsWith(".gz")) bytes = gunzipSync(bytes)
  if (digest(bytes) !== checksums[name]) throw new Error(`Checksum mismatch for ${asset}`)
  await writeFile(`${root}/${name}`, bytes)
}
