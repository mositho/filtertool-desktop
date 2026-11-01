import { readFileSync, writeFileSync } from "node:fs"

export function assertStableVersion(version) {
  if (typeof version !== "string" || version.trim() !== version || !/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)) {
    throw new Error("Use a stable major.minor.patch version, for example 1.0.0")
  }
}

export function setReleaseVersion(root, version) {
  assertStableVersion(version)
  const read = (path) => readFileSync(new URL(path, root), "utf8")
  const pkg = JSON.parse(read("package.json"))
  assertStableVersion(pkg.version)
  const current = pkg.version.split(".").map(BigInt)
  const next = version.split(".").map(BigInt)
  const difference = next.findIndex((part, index) => part !== current[index])
  if (difference !== -1 && next[difference] < current[difference]) {
    throw new Error(`Release version must not be lower than ${pkg.version}`)
  }
  const lock = JSON.parse(read("package-lock.json"))
  if (!lock.packages?.[""]) throw new Error("Expected a root package in package-lock.json")
  const config = read("build/config.yml")
  const pattern = /^  version: "[^"]+"$/gm
  if ([...config.matchAll(pattern)].length !== 1) throw new Error("Expected exactly one product version in build/config.yml")

  pkg.version = version
  lock.version = version
  lock.packages[""].version = version
  const write = (path, contents) => writeFileSync(new URL(path, root), contents)
  write("package.json", JSON.stringify(pkg, null, 2) + "\n")
  write("package-lock.json", JSON.stringify(lock, null, 2) + "\n")
  write("build/config.yml", config.replace(pattern, `  version: "${version}"`))
}
