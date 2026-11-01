import { setReleaseVersion } from "./release-version.mjs"

setReleaseVersion(new URL("../", import.meta.url), process.argv[2])
