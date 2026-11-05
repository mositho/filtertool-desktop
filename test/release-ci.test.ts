// @vitest-environment node
import { spawnSync } from "node:child_process"
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { afterEach, describe, expect, it } from "vitest"

const script = readFileSync(new URL("../scripts/wait-for-release-ci.sh", import.meta.url), "utf8")
const directories: string[] = []
afterEach(() => {
  for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true })
})

// The release coordinator runs on Ubuntu; no GitHub credentials or network needed.
describe.skipIf(process.platform === "win32")("release CI reuse", () => {
  function run({ missing = false, conclusion = "success", watchStatus = 0 } = {}) {
    const directory = mkdtempSync(join(tmpdir(), "filtertool-release-ci-"))
    directories.push(directory)
    writeFileSync(join(directory, "output"), "")
    writeFileSync(join(directory, "sleep"), "#!/bin/sh\nexit 0\n", { mode: 0o755 })
    writeFileSync(
      join(directory, "gh"),
      `#!/bin/bash
set -eu
echo "$*" >> "$CALLS"
case "$1 $2" in
  'run list') ${missing ? ":" : "echo 123"} ;;
  'run watch') exit ${watchStatus} ;;
  'run view') echo '${conclusion}' ;;
  *) exit 99 ;;
esac
`,
      { mode: 0o755 },
    )
    const result = spawnSync("bash", ["-s"], {
      input: script,
      encoding: "utf8",
      timeout: 5000,
      env: {
        ...process.env,
        PATH: `${directory}:${process.env.PATH}`,
        GITHUB_REPOSITORY: "example/filtertool",
        GITHUB_SHA: "exact-release-sha",
        GITHUB_OUTPUT: join(directory, "output"),
        CALLS: join(directory, "calls"),
      },
    })
    return { ...result, output: readFileSync(join(directory, "output"), "utf8"), calls: readFileSync(join(directory, "calls"), "utf8") }
  }

  it("waits for exact-commit main push CI and exposes its successful run", () => {
    const result = run()
    expect(result.status).toBe(0)
    expect(result.output).toBe("run-id=123\n")
    expect(result.calls).toContain("--workflow ci.yml --branch main --event push --commit exact-release-sha")
    expect(result.calls).toContain("run watch 123 --repo example/filtertool --interval 10 --exit-status")
  })

  it("fails instead of using CI from another commit when no matching run exists", () => {
    const result = run({ missing: true })
    expect(result.status).toBe(1)
    expect(result.output).toBe("")
    expect(result.calls).not.toContain("run watch")
  })

  it("does not expose artifacts when waiting for CI fails", () => {
    const result = run({ watchStatus: 1 })
    expect(result.status).toBe(1)
    expect(result.output).toBe("")
  })

  it.each(["failure", "cancelled", "skipped"])("rejects a %s conclusion even if watch exits successfully", (conclusion) => {
    const result = run({ conclusion })
    expect(result.status).toBe(1)
    expect(result.output).toBe("")
  })
})
