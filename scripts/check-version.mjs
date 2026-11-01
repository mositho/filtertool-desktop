import { readFileSync } from "node:fs"
import { assertStableVersion } from "./release-version.mjs"

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8")
const { version } = JSON.parse(read("package.json"))
assertStableVersion(version)
const lock = JSON.parse(read("package-lock.json"))
if (lock.version !== version || lock.packages[""].version !== version) {
  throw new Error("Update package-lock.json to match package.json")
}
const buildVersion = read("build/config.yml").match(/^  version: "([^"]+)"$/m)?.[1]
if (buildVersion !== version) throw new Error("Update build/config.yml to match package.json")
const tag = process.argv[2]
if (tag !== undefined && tag !== `v${version}`) throw new Error(`Release tag must be v${version}`)
