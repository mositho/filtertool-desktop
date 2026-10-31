<div align="center">

<img src="branding/logo.svg" alt="Filtertool" width="80" />

# Filtertool

A Windows item-filter builder for Path of Exile 1, focused on leveling.

[Get started](#get-started) · [Quick guide](#quick-guide) · [Releases](https://github.com/mositho/filtertool-desktop/releases) · [Filter terminology](docs/GLOSSARY.md)

</div>

> [!NOTE]
> The desktop port is in development. There is no stable desktop release yet.

## Get started

Download a stable version from [Releases](https://github.com/mositho/filtertool-desktop/releases):

- **Filtertool-Setup.exe** — install Filtertool for your Windows account.
- **Filtertool.exe** — run without installing. You can put the executable anywhere.

Both versions keep your filters, settings and sounds in **`%APPDATA%\Filtertool`**. Replacing the executable or uninstalling does not remove them. Back up that folder to keep a copy of your work.

Requires 64-bit Windows 10 or 11 and Microsoft Edge WebView2. The installer installs WebView2 if needed; standalone users can get it from [Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/).

Generating TTS sounds also requires **FFmpeg** to be installed separately and available as `ffmpeg.exe` on your `PATH` ([FFmpeg download page](https://ffmpeg.org/download.html)). Filtertool does not bundle, download, or install FFmpeg. If it is not detected, Filtertool will still work, but generating TTS sounds will be unavailable. After installing FFmpeg or changing `PATH`, restart Filtertool.

## Quick guide

1. **Check your game folder.** Filtertool detects your PoE filter path. Use **Browse…** to choose another folder.
2. **Create a filter** in the sidebar. Choose **Caster Template**, **Attack Template**, or a blank starting template, then enter your own filter name and click **Create**.
3. **Save** keeps your configuration locally. It does not update the in-game filter.
4. **Export** saves your configuration and writes the `.filter` file and sound pack to your game folder. **Export All** updates every saved filter. Generating TTS sounds needs FFmpeg (see above) and an internet connection.
5. **Select it in PoE.** Choose the exported filter in the game's item-filter settings. After changes, export again and reload it in-game.

Use **Output Preview** to inspect the generated filter text.

| Want to…                       | Use                                                                    |
| :----------------------------- | :--------------------------------------------------------------------- |
| Bring filters from the web app | **Import → JSON files…** or **Web App folder…**                        |
| Share a filter                 | **Share**, then copy or save the compact JSON                          |
| Include your defaults/styles   | Enable the option in **Share**; imports keep them local to that filter |
| Change defaults or styles      | **Global Settings**                                                    |
| Save / save and export         | `Ctrl+S` / `Ctrl+Shift+S`                                              |
| Undo / redo                    | `Ctrl+Z` / `Ctrl+Shift+Z`                                              |

Import reads existing web JSON without changing the originals. Folder import also preserves JSON defaults/styles and referenced custom MP3s. Deprecated TypeScript-only configurations are not supported. Large shared filters can exceed Discord's message limit; send the JSON file instead.

> [!CAUTION]
> Keep the exported sound folder alongside the `.filter` file. Moving or deleting it will break the filter's sounds. Check your filters after exporting.

[Contributing](docs/CONTRIBUTING.md) · [Third-party notices](docs/THIRD_PARTY.md)
