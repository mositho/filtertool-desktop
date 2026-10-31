import { spawn } from "node:child_process"
import { release } from "node:os"
import { wailsEnvironment } from "./dev-environment.mjs"

const child = spawn("go", ["run", "github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0", ...process.argv.slice(2)], {
  stdio: "inherit",
  env: wailsEnvironment(process.env, process.platform, release()),
})
child.on("error", (error) => {
  console.error(error.message)
  process.exitCode = 1
})
child.on("exit", (code) => {
  process.exitCode = code ?? 1
})
