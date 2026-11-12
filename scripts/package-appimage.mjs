import { spawnSync } from "node:child_process"
import { chmod, copyFile, mkdir, rm, writeFile, rename, access, readFile, stat } from "node:fs/promises"
import { constants } from "node:fs"
import { createHash } from "node:crypto"
import { delimiter, join, resolve } from "node:path"

const run = (command, args, env = {}) => {
  const result = spawnSync(command, args, {
    stdio: "inherit",
    env: { ...process.env, ...env },
  })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`${command} failed with exit code ${result.status ?? 1}`)
}

const appId = "io.github.mositho.filtertool-desktop"
const appDir = resolve("bin/Filtertool.AppDir")
const appImage = resolve("bin/Filtertool-x86_64.AppImage")
const toolVersion = "1.9.1"
const toolSHA256 = "ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0"
const appImageTool = await findAppImageTool()

run("npm", ["run", "build:linux"])

await rm(appDir, { recursive: true, force: true })
await rm(appImage, { force: true })
await mkdir(`${appDir}/usr/bin`, { recursive: true })
await mkdir(`${appDir}/usr/share/icons/hicolor/256x256/apps`, { recursive: true })
await mkdir(`${appDir}/usr/share/applications`, { recursive: true })
await copyFile("bin/Filtertool", `${appDir}/usr/bin/Filtertool`)
await copyFile("branding/logo.png", `${appDir}/usr/share/icons/hicolor/256x256/apps/${appId}.png`)
await copyFile("branding/logo.png", `${appDir}/${appId}.png`)
await writeFile(
  `${appDir}/${appId}.desktop`,
  `[Desktop Entry]\nName=Filtertool\nComment=Path of Exile leveling filter builder\nExec=Filtertool\nIcon=${appId}\nTerminal=false\nType=Application\nCategories=Utility;Game;\nStartupWMClass=${appId}\n`,
)
await copyFile(`${appDir}/${appId}.desktop`, `${appDir}/usr/share/applications/${appId}.desktop`)
await writeFile(`${appDir}/AppRun`, '#!/bin/sh\nHERE="$(dirname "$(readlink -f "$0")")"\nexec "$HERE/usr/bin/Filtertool" "$@"\n')
await chmod(`${appDir}/AppRun`, 0o755)
await chmod(`${appDir}/usr/bin/Filtertool`, 0o755)

run(appImageTool, [appDir, appImage], {
  ARCH: "x86_64",
  APPIMAGE_EXTRACT_AND_RUN: "1",
})
const output = await stat(appImage)
if (!output.isFile() || output.size === 0) throw new Error(`Expected a nonempty AppImage at ${appImage}`)

async function findAppImageTool() {
  if (process.env.APPIMAGETOOL) return process.env.APPIMAGETOOL

  for (const directory of (process.env.PATH || "").split(delimiter)) {
    const candidate = join(directory, "appimagetool")
    try {
      await access(candidate, constants.X_OK)
      return candidate
    } catch {
      // Try the next PATH entry.
    }
  }

  const directory = resolve("bin")
  const candidate = join(directory, `appimagetool-${toolVersion}`)
  try {
    verifyTool(await readFile(candidate))
    return candidate
  } catch (error) {
    if (error.code !== "ENOENT") throw error
    // Download a checksum-pinned local copy if it is not already cached.
  }

  await mkdir(directory, { recursive: true })
  const temporary = `${candidate}.download`
  try {
    const response = await fetch(`https://github.com/AppImage/appimagetool/releases/download/${toolVersion}/appimagetool-x86_64.AppImage`, {
      signal: AbortSignal.timeout(120_000),
    })
    if (!response.ok) throw new Error(`HTTP ${response.status} ${response.statusText}`)
    const contents = Buffer.from(await response.arrayBuffer())
    verifyTool(contents)
    await writeFile(temporary, contents)
    await chmod(temporary, 0o755)
    await rename(temporary, candidate)
    return candidate
  } catch (error) {
    await rm(temporary, { force: true })
    throw new Error(`Could not download appimagetool. Install it on PATH or set APPIMAGETOOL to its path. ${error.message}`)
  }
}

function verifyTool(contents) {
  if (createHash("sha256").update(contents).digest("hex") !== toolSHA256) {
    throw new Error(`appimagetool ${toolVersion} checksum mismatch`)
  }
}
