// @vitest-environment node
import { describe, expect, it } from "vitest"
import { wailsEnvironment } from "../scripts/dev-environment.mjs"

describe("WSL development launcher", () => {
  it("forwards the Wails URL without changing unrelated WSLENV entries", () => {
    const environment = { WSL_DISTRO_NAME: "Ubuntu", WSLENV: "WT_SESSION:MY_PATH/p", OTHER: "keep" }
    expect(wailsEnvironment(environment, "linux", "6.6.0-microsoft-standard-WSL2")).toEqual({
      ...environment,
      CGO_ENABLED: "0",
      WSLENV: "WT_SESSION:MY_PATH/p:FRONTEND_DEVSERVER_URL/w",
    })
    expect(environment.WSLENV).toBe("WT_SESSION:MY_PATH/p")
  })

  it("does not path-translate the URL or duplicate an existing entry", () => {
    expect(wailsEnvironment({ WSLENV: "FRONTEND_DEVSERVER_URL/up:WT_SESSION" }, "linux", "microsoft").WSLENV).toBe(
      "WT_SESSION:FRONTEND_DEVSERVER_URL/w",
    )
  })

  it.each(["win32", "darwin", "linux"])("leaves non-WSL %s environments alone", (platform) => {
    expect(wailsEnvironment({ PATH: "unchanged" }, platform, "ordinary-kernel")).toEqual({ PATH: "unchanged", CGO_ENABLED: "0" })
  })
})
