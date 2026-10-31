import { spawn } from "node:child_process"
import { release } from "node:os"
import { wailsEnvironment } from "./dev-environment.mjs"

const args = process.argv.slice(2)
const environment = { ...process.env }
if (args.includes("--templates")) environment.FILTERTOOL_PROFILE = "templates"
const child = spawn("go", ["run", "github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0", ...args.filter((arg) => arg !== "--templates")], {
  stdio: "inherit",
  env: wailsEnvironment(environment, process.platform, release()),
})
child.on("error", (error) => {
  console.error(error.message)
  process.exitCode = 1
})
child.on("exit", (code) => {
  process.exitCode = code ?? 1
})
