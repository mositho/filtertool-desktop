// @vitest-environment node
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { pathToFileURL } from "node:url"
import { afterEach, describe, expect, it } from "vitest"
import { assertStableVersion, setReleaseVersion } from "../scripts/release-version.mjs"

const directories: string[] = []
afterEach(() => {
  for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true })
})

function fixture() {
  const directory = mkdtempSync(join(tmpdir(), "filtertool-release-"))
  directories.push(directory)
  mkdirSync(join(directory, "build"))
  writeFileSync(join(directory, "package.json"), JSON.stringify({ name: "test", version: "1.0.0", private: true }))
  writeFileSync(
    join(directory, "package-lock.json"),
    JSON.stringify({ version: "1.0.0", packages: { "": { version: "1.0.0" }, "dependency": { version: "9.0.0" } } }),
  )
  writeFileSync(join(directory, "build/config.yml"), 'version: "3"\ninfo:\n  version: "1.0.0"\n')
  return {
    root: pathToFileURL(directory + "/"),
    read: (path: string) => readFileSync(join(directory, path), "utf8"),
    write: (path: string, contents: string) => writeFileSync(join(directory, path), contents),
  }
}

describe("release version preparation", () => {
  it.each(["1.0.1", "1.1.0", "2.0.0", "10.0.0"])("updates product metadata to %s without changing dependencies", (version) => {
    const { root, read } = fixture()
    setReleaseVersion(root, version)
    expect(JSON.parse(read("package.json"))).toEqual({ name: "test", version, private: true })
    expect(JSON.parse(read("package-lock.json"))).toEqual({ version, packages: { "": { version }, "dependency": { version: "9.0.0" } } })
    expect(read("build/config.yml")).toBe(`version: "3"\ninfo:\n  version: "${version}"\n`)
  })

  it("allows the current version and is idempotent", () => {
    const { root, read } = fixture()
    setReleaseVersion(root, "1.0.0")
    const before = read("package.json")
    setReleaseVersion(root, "1.0.0")
    expect(read("package.json")).toBe(before)
  })

  it.each([undefined, "", "v1.0.0", "1.0.0-beta.1", "1.0.0+build", "01.0.0", "1.0", "1.0.0\n", "1.0.0; echo unsafe"])(
    "rejects invalid input %j without touching files",
    (version) => {
      const { root, read } = fixture()
      const before = read("package.json")
      expect(() => setReleaseVersion(root, version)).toThrow("stable major.minor.patch")
      expect(read("package.json")).toBe(before)
    },
  )

  it.each(["0.9.9", "0.99.99"])("rejects a downgrade to %s", (version) => {
    const { root } = fixture()
    expect(() => setReleaseVersion(root, version)).toThrow("must not be lower")
  })

  it("compares versions numerically rather than alphabetically", () => {
    const { root } = fixture()
    setReleaseVersion(root, "1.10.0")
    expect(() => setReleaseVersion(root, "1.9.9")).toThrow("must not be lower")
  })

  it("fails before writing if the build metadata is missing or ambiguous", () => {
    const { root, read, write } = fixture()
    const before = read("package.json")
    for (const config of ['version: "3"\n', 'info:\n  version: "1.0.0"\n  version: "1.0.1"\n']) {
      write("build/config.yml", config)
      expect(() => setReleaseVersion(root, "1.1.0")).toThrow("exactly one product version")
      expect(read("package.json")).toBe(before)
    }
  })

  it("fails before writing if the lockfile has no root package", () => {
    const { root, read, write } = fixture()
    const before = read("package.json")
    write("package-lock.json", '{"version":"1.0.0"}')
    expect(() => setReleaseVersion(root, "1.1.0")).toThrow("root package")
    expect(read("package.json")).toBe(before)
  })

  it("validates stable versions independently of metadata", () => {
    expect(() => assertStableVersion("1.0.0")).not.toThrow()
    expect(() => assertStableVersion("1.0.0-rc.1")).toThrow()
  })
})
