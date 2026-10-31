// @vitest-environment node
import { readFileSync } from "node:fs"
import { expect, it } from "vitest"

it("credits mositho in package and Windows publisher metadata", () => {
  const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8")
  expect(JSON.parse(read("package.json")).author).toBe("mositho")
  expect(read("scripts/build.mjs")).toContain('CompanyName: "mositho"')
  expect(read("build/config.yml")).toContain('companyName: "mositho"')
  expect(read("build/windows/installer.nsi")).toMatch(/"Publisher"\s+"mositho"/)
})
