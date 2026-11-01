// @vitest-environment node
import { readFileSync } from "node:fs"
import { spawnSync } from "node:child_process"
import { expect, it } from "vitest"

it("credits mositho in package and Windows publisher metadata", () => {
  const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8")
  expect(JSON.parse(read("package.json")).author).toBe("mositho")
  expect(read("scripts/build.mjs")).toContain('CompanyName: "mositho"')
  expect(read("build/config.yml")).toContain('companyName: "mositho"')
  expect(read("build/windows/installer.nsi")).toMatch(/"Publisher"\s+"mositho"/)
})

it("keeps stable release versions consistent and rejects mismatched or prerelease tags", () => {
  const root = new URL("../", import.meta.url)
  const { version } = JSON.parse(readFileSync(new URL("package.json", root), "utf8"))
  const check = (tag?: string) =>
    spawnSync(process.execPath, ["scripts/check-version.mjs", ...(tag === undefined ? [] : [tag])], { cwd: root, encoding: "utf8" })
  expect(check().status).toBe(0)
  expect(check(`v${version}`).status).toBe(0)
  expect(check(`v${version}-beta.1`).status).not.toBe(0)
  expect(check("v999.0.0").status).not.toBe(0)
})
