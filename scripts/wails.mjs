import { spawn } from "node:child_process"
import { release } from "node:os"
import { fileURLToPath } from "node:url"
import { wailsEnvironment } from "./dev-environment.mjs"

const args = process.argv.slice(2)
const environment = { ...process.env }
if (args.includes("--dev-profile")) {
  environment.FILTERTOOL_PROFILE = "dev"
  environment.FILTERTOOL_SOURCE_DIR = fileURLToPath(new URL("../internal/data/", import.meta.url))
}
const child = spawn("go", ["run", "github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0", ...args.filter((arg) => arg !== "--dev-profile")], {
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
