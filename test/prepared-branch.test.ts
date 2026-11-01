// @vitest-environment node
import { execFileSync, spawnSync } from "node:child_process"
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { expect, it } from "vitest"

// Privileged proposal jobs run on Ubuntu only; Windows CI still checks workflow structure.
it.skipIf(process.platform === "win32")("pushes only prepared metadata, bypasses hooks, and never overwrites a proposal", () => {
  const directory = mkdtempSync(join(tmpdir(), "filtertool-proposal-"))
  try {
    const remote = join(directory, "remote.git")
    const cwd = join(directory, "checkout")
    mkdirSync(cwd)
    const git = (...args: string[]) => execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim()
    git("init", "--bare", remote)
    git("init", "-b", "main")
    git("config", "user.name", "Test")
    git("config", "user.email", "test@example.invalid")
    mkdirSync(join(cwd, "build"))
    mkdirSync(join(cwd, "prepared/build"), { recursive: true })
    const paths = ["package.json", "package-lock.json", "build/config.yml"]
    for (const path of paths) {
      writeFileSync(join(cwd, path), "original\n")
      writeFileSync(join(cwd, "prepared", path), "prepared\n")
    }
    writeFileSync(join(cwd, "prepared/unexpected.txt"), "must not be committed\n")
    git("add", "--", ...paths)
    git("-c", "core.hooksPath=/dev/null", "commit", "-m", "fixture")
    const base = git("rev-parse", "HEAD")
    git("remote", "add", "origin", remote)
    git("push", "origin", "main")
    writeFileSync(join(cwd, ".git/hooks/pre-commit"), "#!/bin/sh\nexit 99\n", { mode: 0o755 })
    const summary = join(directory, "summary.md")
    const script = fileURLToPath(new URL("../scripts/push-prepared-branch.sh", import.meta.url))
    const run = (version: string) =>
      spawnSync("bash", [script, "release"], {
        cwd,
        encoding: "utf8",
        env: {
          ...process.env,
          GH_TOKEN: "test-token",
          GITHUB_REPOSITORY: "test/repo",
          GITHUB_STEP_SUMMARY: summary,
          RELEASE_VERSION: version,
        },
      })
    const result = run("1.1.0")
    expect(result.status, result.stderr).toBe(0)
    const proposed = git("rev-parse", "HEAD")
    expect(git("ls-remote", "origin", "refs/heads/main").split(/\s/)[0]).toBe(base)
    expect(git("ls-remote", "origin", "refs/heads/release/v1.1.0").split(/\s/)[0]).toBe(proposed)
    expect(git("diff", "--name-only", base, proposed).split("\n").sort()).toEqual([...paths].sort())
    for (const path of paths) expect(git("show", `${proposed}:${path}`)).toBe("prepared")
    expect(readFileSync(summary, "utf8")).toContain("/compare/main...release/v1.1.0?expand=1")

    writeFileSync(join(cwd, "prepared/package.json"), "different\n")
    expect(run("1.1.0").status).toBe(0)
    expect(git("rev-parse", "HEAD")).toBe(proposed)
    expect(readFileSync(summary, "utf8")).toContain("Existing branch left untouched")
    expect(run("1.1.0; echo unsafe").status).not.toBe(0)
    expect(git("rev-parse", "HEAD")).toBe(proposed)
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
})
