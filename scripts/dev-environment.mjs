// Windows executables launched from WSL do not inherit Linux environment
// variables unless WSLENV names them. Wails supplies the URL after startup.
export function wailsEnvironment(environment, platform, osRelease) {
  const result = { ...environment, CGO_ENABLED: "0" }
  const wsl = platform === "linux" && (environment.WSL_DISTRO_NAME || /microsoft/i.test(osRelease))
  if (wsl) {
    const entries = (environment.WSLENV ?? "").split(":").filter(Boolean)
    // Replace flags on this one entry: a URL must not undergo path translation.
    result.WSLENV = [...entries.filter((entry) => entry.split("/")[0] !== "FRONTEND_DEVSERVER_URL"), "FRONTEND_DEVSERVER_URL/w"].join(":")
    if (environment.FILTERTOOL_PROFILE) {
      result.WSLENV = [
        ...result.WSLENV.split(":").filter((entry) => entry.split("/")[0] !== "FILTERTOOL_PROFILE"),
        "FILTERTOOL_PROFILE/w",
      ].join(":")
    }
    if (environment.FILTERTOOL_SOURCE_DIR) {
      result.WSLENV = [
        ...result.WSLENV.split(":").filter((entry) => entry.split("/")[0] !== "FILTERTOOL_SOURCE_DIR"),
        "FILTERTOOL_SOURCE_DIR/pw",
      ].join(":")
    }
  }
  return result
}
