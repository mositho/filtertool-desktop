// @vitest-environment node
import { readFileSync } from "node:fs"
import { spawnSync } from "node:child_process"
import { expect, it } from "vitest"

it("does not retain the unused PrimeVue license-managed UI stack", () => {
  const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8")
  const pkg = JSON.parse(read("package.json"))
  const lock = JSON.parse(read("package-lock.json"))
  expect(pkg.dependencies).not.toHaveProperty("primevue")
  expect(read("frontend/src/main.ts")).not.toContain("primevue")
  for (const path of Object.keys(lock.packages)) {
    expect(path).not.toMatch(/node_modules\/(?:primevue(?:\/|$)|@prime(?:vue|ui|uix|icons)\/)/)
  }
})

it("keeps the Node runtime, package engines and type definitions on the same major", () => {
  const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8")
  const major = Number(read(".nvmrc").trim())
  const pkg = JSON.parse(read("package.json"))
  const lock = JSON.parse(read("package-lock.json"))
  expect(Number.isSafeInteger(major) && major > 0).toBe(true)
  expect(pkg.engines.node).toBe(`>=${major} <${major + 1}`)
  expect(pkg.devDependencies["@types/node"]).toMatch(new RegExp(`^\\^${major}\\.`))
  expect(lock.packages[""].engines.node).toBe(pkg.engines.node)
  expect(lock.packages[""].devDependencies["@types/node"]).toBe(pkg.devDependencies["@types/node"])
  expect(lock.packages["node_modules/@types/node"].version.split(".")[0]).toBe(String(major))
})

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
