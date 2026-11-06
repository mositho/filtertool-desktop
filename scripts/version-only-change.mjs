import { execFileSync } from "node:child_process"
import { appendFileSync } from "node:fs"
import { pathToFileURL } from "node:url"
import { isDeepStrictEqual } from "node:util"
import { assertStableVersion } from "./release-version.mjs"

const metadata = ["build/config.yml", "package-lock.json", "package.json"]

// Only skip expensive PR checks when the entire diff is a consistent version bump.
// Changes to dependencies, scripts, build settings or any other file use full CI.
export function isVersionOnlyChange(paths, before, after) {
  if (!isDeepStrictEqual([...paths].sort(), metadata)) return false
  try {
    const basePackage = JSON.parse(before("package.json"))
    const nextPackage = JSON.parse(after("package.json"))
    assertStableVersion(basePackage.version)
    assertStableVersion(nextPackage.version)
    const previous = basePackage.version.split(".").map(BigInt)
    const next = nextPackage.version.split(".").map(BigInt)
    const difference = next.findIndex((value, index) => value !== previous[index])
    if (difference === -1 || next[difference] < previous[difference]) return false

    const versionPattern = /^  version: "([^"]+)"$/gm
    for (const read of [before, after]) {
      const { version } = JSON.parse(read("package.json"))
      const lock = JSON.parse(read("package-lock.json"))
      const matches = [...read("build/config.yml").matchAll(versionPattern)]
      if (lock.version !== version || lock.packages?.[""]?.version !== version || matches.length !== 1 || matches[0][1] !== version) {
        return false
      }
    }
    basePackage.version = nextPackage.version
    if (!isDeepStrictEqual(basePackage, nextPackage)) return false
    const baseLock = JSON.parse(before("package-lock.json"))
    baseLock.version = nextPackage.version
    baseLock.packages[""].version = nextPackage.version
    return (
      isDeepStrictEqual(baseLock, JSON.parse(after("package-lock.json"))) &&
      before("build/config.yml").replace(versionPattern, `  version: "${nextPackage.version}"`) === after("build/config.yml")
    )
  } catch {
    return false
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const { BASE_SHA: base, HEAD_SHA: head, GITHUB_OUTPUT: output } = process.env
  if (![base, head].every((sha) => /^[a-f0-9]{40,64}$/.test(sha ?? ""))) throw new Error("Expected base and head commit SHAs")
  const git = (...args) => execFileSync("git", args, { encoding: "utf8" })
  const paths = git("diff", "--name-only", "--no-renames", "-z", base, head).split("\0").filter(Boolean)
  const result = isVersionOnlyChange(
    paths,
    (path) => git("show", `${base}:${path}`),
    (path) => git("show", `${head}:${path}`),
  )
  appendFileSync(output, `version-only=${result}\n`)
  console.log(result ? "Verified version-only change; native CI will run after merge." : "Not a version-only change; full CI required.")
}
