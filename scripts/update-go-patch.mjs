import { readFileSync, writeFileSync } from "node:fs"

// Only update the project's existing Go release line, never a minor or prerelease.
export function updateGoPatch(contents, releases) {
  const current = /^go (\d+)\.(\d+)\.(\d+)$/m.exec(contents)
  if (!current) throw new Error("Expected a stable major.minor.patch Go directive in go.mod")
  let patch = Number(current[3])
  for (const release of releases) {
    const version = /^go(\d+)\.(\d+)\.(\d+)$/.exec(release.version)
    if (release.stable && version && version[1] === current[1] && version[2] === current[2]) {
      patch = Math.max(patch, Number(version[3]))
    }
  }
  return contents.replace(current[0], `go ${current[1]}.${current[2]}.${patch}`)
}

if (import.meta.main) {
  const response = await fetch("https://go.dev/dl/?mode=json&include=all", { signal: AbortSignal.timeout(30_000) })
  if (!response.ok) throw new Error(`Go release lookup failed: HTTP ${response.status}`)
  const contents = readFileSync("go.mod", "utf8")
  const updated = updateGoPatch(contents, await response.json())
  if (updated !== contents) writeFileSync("go.mod", updated)
}
