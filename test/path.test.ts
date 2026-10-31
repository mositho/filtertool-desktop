import { expect, it } from "vitest"
import { pruneEmpty } from "../frontend/src/path"

it("retains deliberate empty selections and null style overrides", () => {
  expect(pruneEmpty({ preferredColors: [], border: null, residue: {}, unset: undefined })).toEqual({ preferredColors: [], border: null })
})
