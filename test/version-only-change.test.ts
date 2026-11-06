// @vitest-environment node
import { execFileSync, spawnSync } from "node:child_process"
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { describe, expect, it } from "vitest"
import { isVersionOnlyChange } from "../scripts/version-only-change.mjs"

function metadata(version: string): Record<string, string> {
  return {
    "package.json": JSON.stringify({ version, scripts: { check: "npm test" } }),
    "package-lock.json": JSON.stringify({ version, packages: { "": { version }, "node_modules/example": { version: "3.0.0" } } }),
    "build/config.yml": `version: "3"\ninfo:\n  version: "${version}"\n  name: Filtertool\n`,
  }
}

describe("version-only CI fast path", () => {
  const before = metadata("1.0.0")
  function check(after: Record<string, string>, paths = Object.keys(after)) {
    return isVersionOnlyChange(
      paths,
      (path: string) => before[path],
      (path: string) => after[path],
    )
  }

  it("accepts a consistent version bump without relying on a branch name", () => {
    expect(check(metadata("1.0.1"))).toBe(true)
    expect(check(metadata("2.0.0"))).toBe(true)
  })

  it("requires full CI for any additional changed file", () => {
    expect(check({ ...metadata("1.0.1"), "frontend/src/App.vue": "changed" })).toBe(false)
  })

  it.each(["script", "dependency", "build setting"])("requires full CI for a %s change inside version metadata", (change) => {
    const after = metadata("1.0.1")
    if (change === "script") {
      const pkg = JSON.parse(after["package.json"])
      pkg.scripts.check = "echo skipped"
      after["package.json"] = JSON.stringify(pkg)
    } else if (change === "dependency") {
      const lock = JSON.parse(after["package-lock.json"])
      lock.packages["node_modules/example"].version = "4.0.0"
      after["package-lock.json"] = JSON.stringify(lock)
    } else {
      after["build/config.yml"] += "extra: changed\n"
    }
    expect(check(after)).toBe(false)
  })

  it.each(["1.0.0", "0.9.0", "1.0.1-beta.1"])("requires full CI for non-increasing or unstable version %s", (version) => {
    expect(check(metadata(version))).toBe(false)
  })

  it("requires full CI for mismatched, missing or malformed metadata", () => {
    const after = metadata("1.0.1")
    expect(check({ ...after, "package-lock.json": before["package-lock.json"] })).toBe(false)
    expect(check({ ...after, "package.json": "not JSON" })).toBe(false)
    expect(check({ ...after, "build/config.yml": after["build/config.yml"] + '  version: "1.0.1"\n' })).toBe(false)
    expect(check(after, ["package.json"])).toBe(false)
  })

  it("classifies actual commit diffs and writes the GitHub Actions output", () => {
    const directory = mkdtempSync(join(tmpdir(), "filtertool-version-only-"))
    try {
      const git = (...args: string[]) => execFileSync("git", args, { cwd: directory, encoding: "utf8" }).trim()
      const writeMetadata = (version: string) => {
        for (const [path, contents] of Object.entries(metadata(version))) writeFileSync(join(directory, path), contents)
      }
      mkdirSync(join(directory, "build"))
      git("init", "--quiet")
      git("config", "user.name", "Test")
      git("config", "user.email", "test@example.invalid")
      writeMetadata("1.0.0")
      git("add", ".")
      git("commit", "--quiet", "-m", "base")
      const base = git("rev-parse", "HEAD")
      writeMetadata("1.0.1")
      git("add", ".")
      git("commit", "--quiet", "-m", "version bump")

      for (const extraChange of [false, true]) {
        if (extraChange) {
          writeFileSync(join(directory, "application.go"), "package main\n")
          git("add", "application.go")
          git("commit", "--quiet", "-m", "application change")
        }
        const output = join(directory, "actions-output")
        writeFileSync(output, "")
        const result = spawnSync(process.execPath, [fileURLToPath(new URL("../scripts/version-only-change.mjs", import.meta.url))], {
          cwd: directory,
          encoding: "utf8",
          env: { ...process.env, BASE_SHA: base, HEAD_SHA: git("rev-parse", "HEAD"), GITHUB_OUTPUT: output },
        })
        expect(result.status, result.stderr).toBe(0)
        expect(readFileSync(output, "utf8")).toBe(`version-only=${!extraChange}\n`)
      }
    } finally {
      rmSync(directory, { recursive: true, force: true })
    }
  })
})
