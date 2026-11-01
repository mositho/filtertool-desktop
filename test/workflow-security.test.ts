// @vitest-environment node
import { readFileSync, readdirSync } from "node:fs"
import { expect, it } from "vitest"
import { parse } from "yaml"

const root = new URL("../", import.meta.url)
const read = (path: string) => readFileSync(new URL(path, root), "utf8")
const workflows = readdirSync(new URL(".github/workflows/", root))
  .filter((name) => name.endsWith(".yml"))
  .map((name) => ({ name, config: parse(read(`.github/workflows/${name}`)) }))

it("defaults all workflows to read-only and pins every allowlisted external action", () => {
  const policy = JSON.parse(read(".github/actions-policy.json"))
  expect(policy.github_owned_allowed).toBe(false)
  expect(policy.verified_allowed).toBe(false)
  for (const { config } of workflows) {
    expect(config.permissions).toEqual({ contents: "read" })
    expect(config.on).not.toHaveProperty("pull_request_target")
    for (const job of Object.values(config.jobs) as any[]) {
      for (const step of job.steps ?? []) {
        if (step.uses) {
          expect(step.uses).toMatch(/^[\w-]+\/[\w-]+@[a-f0-9]{40}$/)
          expect(policy.patterns_allowed).toContain(step.uses.split("@")[0] + "@*")
        }
        if (step.uses?.startsWith("actions/checkout@")) expect(step.with?.["persist-credentials"]).toBe(false)
        if (step.run?.includes("npm ci")) expect(step.run).toContain("npm ci --ignore-scripts")
      }
    }
  }
})

it("keeps installation/builds out of fresh write-enabled jobs", () => {
  for (const { config } of workflows) {
    for (const job of Object.values(config.jobs) as any[]) {
      if (job.permissions?.contents !== "write") continue
      expect(job["runs-on"]).toBe("ubuntu-latest")
      expect(job.permissions).toEqual({ contents: "write" })
      expect(job.needs).toBeDefined()
      expect(job.steps.some((step: any) => step.uses?.startsWith("actions/download-artifact@"))).toBe(true)
      for (const step of job.steps) {
        expect(step.uses ?? "").not.toMatch(/actions\/setup-(node|go)/)
        expect(step.run ?? "").not.toMatch(/\b(npm|npx|node|go|choco|makensis)\b|package-windows\.ps1/)
      }
    }
  }
})

it("keeps releases manual, conditional on checked metadata, and off protected main", () => {
  const release = parse(read(".github/workflows/release.yml"))
  expect(Object.keys(release.on)).toEqual(["workflow_dispatch"])
  expect(release.concurrency).toEqual({ "group": "release", "cancel-in-progress": false })
  expect(release.jobs.checks.if).toBe("needs.validate.outputs.changed == 'false'")
  expect(release.jobs.package.needs).toContain("checks")
  expect(release.jobs.package.permissions).toBeUndefined()
  expect(release.jobs.draft.needs).toContain("package")
  expect(read(".github/workflows/release.yml")).toContain("--draft --verify-tag")
  expect(read(".github/workflows/release.yml")).not.toContain("HEAD:refs/heads/main")
  expect(read("scripts/push-prepared-branch.sh")).not.toMatch(/git push[^\n]*(--force|refs\/heads\/main)|gh pr (create|merge|review)/)
})

it("disables unattended merging and gives routine updates a seven-day cooldown", () => {
  expect(workflows.map(({ name }) => name)).not.toContain("dependabot-automerge.yml")
  const dependabot = parse(read(".github/dependabot.yml"))
  for (const update of dependabot.updates) expect(update.cooldown).toEqual({ "default-days": 7 })
})
