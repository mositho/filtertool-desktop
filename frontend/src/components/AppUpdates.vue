<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { Browser } from "@wailsio/runtime"
import { api, type UpdateStatus } from "../api"

const props = defineProps<{ unsaved: boolean; busy: boolean; target?: string }>()
const status = ref<UpdateStatus>({ enabled: true, version: "", ready: false })
const checking = ref(false)
const downloading = ref(false)
const restarting = ref(false)
const open = ref(false)
const acceptedMajor = ref(false)
const error = ref("")
const message = ref("")
const working = computed(() => checking.value || downloading.value || restarting.value)

async function check(manual = false) {
  if (working.value) return
  checking.value = true
  if (manual) open.value = true
  error.value = ""
  message.value = ""
  try {
    const next = await api.checkUpdates()
    if (next.release?.version !== status.value.release?.version) acceptedMajor.value = false
    status.value = next
    if (manual) {
      if (!next.release) message.value = "You're using the latest published version."
    }
  } catch (e) {
    if (manual) {
      error.value = `Could not check for updates: ${(e as Error).message}`
    }
    // Offline startup and GitHub rate limits must not interrupt editing.
  } finally {
    checking.value = false
  }
}

async function download() {
  const release = status.value.release
  if (!release || working.value || (release.major && !acceptedMajor.value)) return
  downloading.value = true
  error.value = ""
  try {
    status.value = await api.downloadUpdate(release.version, acceptedMajor.value)
  } catch (e) {
    error.value = `Update failed. Your current application has not been replaced. ${(e as Error).message}`
  } finally {
    downloading.value = false
  }
}

async function restart() {
  if (props.unsaved || props.busy || working.value) return
  restarting.value = true
  error.value = ""
  try {
    await api.setDirty(props.unsaved)
    await api.restartUpdate()
  } catch (e) {
    error.value = `Could not restart: ${(e as Error).message}`
    restarting.value = false
  }
}

async function releaseNotes() {
  try {
    await Browser.OpenURL(
      `https://github.com/mositho/filtertool-desktop/releases/tag/v${encodeURIComponent(status.value.release!.version)}`,
    )
  } catch (e) {
    error.value = (e as Error).message
  }
}

onMounted(() => void check())
</script>

<template>
  <Teleport :to="target ?? 'body'" :disabled="!target" defer>
    <button
      v-if="status.enabled"
      type="button"
      class="flex w-full flex-col items-center gap-1 border-t border-neutral-800 px-3 py-3 text-center text-xs text-neutral-400 hover:bg-neutral-800 hover:text-white focus-visible:outline focus-visible:outline-green-600 disabled:opacity-50"
      :disabled="!status.release && working"
      @click="status.release ? (open = true) : check(true)"
    >
      <span>Filtertool{{ status.version ? ` ${status.version}` : "" }}</span>
      <span v-if="status.release" class="font-medium" :class="status.release.major ? 'text-amber-300' : 'text-blue-300'">
        {{ status.ready ? "Update ready — restart to apply" : `Version ${status.release.version} available` }}
        {{ status.release.major ? " · Major update: breaking changes possible" : "" }}
      </span>
      <span v-else>
        {{ checking ? "Checking for updates…" : "Check for updates" }}
      </span>
    </button>
  </Teleport>

  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6" @click.self="open = false">
    <section
      role="dialog"
      aria-modal="true"
      aria-labelledby="update-title"
      class="max-h-[85vh] w-full max-w-xl overflow-y-auto rounded-lg border border-neutral-700 bg-neutral-900 p-6 shadow-xl"
    >
      <h2 id="update-title" class="mb-3 text-lg font-semibold">Filtertool updates</h2>
      <p v-if="checking" role="status" class="mb-3 text-sm text-neutral-300">Checking for updates…</p>
      <p v-if="status.version" class="mb-3 text-sm text-neutral-400">
        Installed: {{ status.version }}<template v-if="status.release"> → Available: {{ status.release.version }}</template>
      </p>
      <div
        v-if="status.release?.major"
        role="alert"
        class="mb-4 rounded border border-amber-700 bg-amber-950/50 p-4 text-sm text-amber-200"
      >
        <strong>Major update — potentially breaking changes</strong>
        <p class="mt-2">Your existing filters may need to be migrated or recreated. Automatic migration is not provided.</p>
        <p class="mt-2">
          Before updating, close Filtertool and back up the entire <code>%APPDATA%\Filtertool</code> folder. Read the release notes for
          migration instructions.
        </p>
      </div>
      <template v-if="status.release">
        <button type="button" class="mb-3 text-sm text-blue-300 hover:underline" @click="releaseNotes">
          Read release notes on GitHub ↗
        </button>
        <pre
          class="mb-4 max-h-52 overflow-auto whitespace-pre-wrap break-words rounded bg-neutral-950 p-3 font-sans text-sm text-neutral-300"
          >{{ status.release.notes || "No release notes provided. Review the release on GitHub before updating." }}</pre>
        <label v-if="status.release.major && !status.ready" class="mb-4 flex items-start gap-2 text-sm text-amber-200">
          <input v-model="acceptedMajor" type="checkbox" class="mt-1" :disabled="working" />
          I have backed up my data and understand that my filters may need migration or recreation.
        </label>
        <p v-if="downloading" role="status" class="mb-3 text-sm text-blue-300">
          Downloading and verifying update… You can return to editing, but keep Filtertool open.
        </p>
        <p v-if="status.ready" class="mb-3 text-sm text-green-300">Update downloaded and verified. Restart to apply it.</p>
        <p v-if="status.ready && (unsaved || busy)" role="status" class="mb-3 text-sm text-amber-200">
          Save or discard unsaved filter/global changes and wait for current operations to finish before restarting. Choose Return to go
          back to editing.
        </p>
      </template>
      <p v-if="message" role="status" class="mb-3 text-sm text-neutral-300">{{ message }}</p>
      <p v-if="error" role="alert" class="mb-3 break-words text-sm text-red-300">{{ error }}</p>
      <div class="mt-4 flex justify-end gap-2">
        <button
          type="button"
          class="rounded bg-neutral-800 px-3 py-2 text-sm hover:bg-neutral-700 disabled:opacity-50"
          @click="open = false"
        >
          Return
        </button>
        <button
          v-if="!status.release"
          type="button"
          class="rounded bg-neutral-800 px-3 py-2 text-sm disabled:opacity-50"
          :disabled="working"
          @click="check(true)"
        >
          Check again
        </button>
        <button
          v-else-if="!status.ready"
          type="button"
          class="rounded bg-blue-700 px-3 py-2 text-sm hover:bg-blue-600 disabled:opacity-50"
          :disabled="working || (status.release.major && !acceptedMajor)"
          @click="download"
        >
          {{ downloading ? "Downloading…" : "Download update" }}
        </button>
        <button
          v-else
          type="button"
          class="rounded bg-blue-700 px-3 py-2 text-sm hover:bg-blue-600 disabled:opacity-50"
          :disabled="working || unsaved || busy"
          @click="restart"
        >
          {{ restarting ? "Restarting…" : "Restart & Apply" }}
        </button>
      </div>
    </section>
  </div>
</template>
