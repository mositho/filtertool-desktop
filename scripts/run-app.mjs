import { spawn } from "node:child_process"
import { release } from "node:os"

const wsl = process.platform === "linux" && (process.env.WSL_DISTRO_NAME || /microsoft/i.test(release()))
const executable = process.platform === "win32" || wsl ? "./bin/Filtertool.exe" : "./bin/Filtertool"
const child = spawn(executable, [], { stdio: "inherit" })
child.on("error", (error) => {
  console.error(error.message)
  process.exitCode = 1
})
child.on("exit", (code) => {
  process.exitCode = code ?? 1
})
