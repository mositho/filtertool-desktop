import { spawnSync } from "node:child_process"
import { mkdir, readFile, writeFile, rm } from "node:fs/promises"

const dev = process.argv.includes("--dev")
const run = (command, args, env = {}) => {
  const result = spawnSync(command, args, {
    stdio: "inherit",
    env: { ...process.env, ...env },
    shell: process.platform === "win32" && command === "npm",
  })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`${command} failed with exit code ${result.status ?? 1}`)
}
run("node", ["scripts/generate-branding.mjs"])
run("npm", ["run", "frontend:build"])
await mkdir("bin", { recursive: true })
const { version } = JSON.parse(await readFile("package.json", "utf8"))
const info = {
  fixed: { file_version: version, product_version: version },
  info: {
    "0409": {
      ProductName: "Filtertool",
      FileDescription: "Filtertool",
      CompanyName: "mositho",
      FileVersion: version,
      ProductVersion: version,
      InternalName: "filtertool-desktop",
      OriginalFilename: "Filtertool.exe",
    },
  },
}
await writeFile("build/windows/info.json", JSON.stringify(info, null, 2) + "\n")
run("node", [
  "scripts/wails.mjs",
  "generate",
  "syso",
  "-arch",
  "amd64",
  "-icon",
  "build/windows/icon.ico",
  "-manifest",
  "build/windows/app.manifest",
  "-info",
  "build/windows/info.json",
  "-out",
  "filtertool_windows_amd64.syso",
])
try {
  run(
    "go",
    [
      "build",
      "-buildvcs=false",
      ...(dev ? [] : ["-tags", "production", "-trimpath"]),
      "-ldflags",
      "-s -w -H windowsgui",
      "-o",
      "bin/Filtertool.exe",
      ".",
    ],
    { GOOS: "windows", GOARCH: "amd64", CGO_ENABLED: "0" },
  )
} finally {
  await rm("filtertool_windows_amd64.syso", { force: true })
}
