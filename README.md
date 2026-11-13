<div align="center">

<img src="branding/logo.svg" alt="Filtertool" width="80" />

# Filtertool

A desktop item-filter builder for Path of Exile 1, focused on leveling.

<p>
  <a href="https://github.com/mositho/filtertool-desktop/releases/latest/download/Filtertool-Setup.exe"><img src="https://img.shields.io/badge/Download-Windows%20Installer-2563eb?style=for-the-badge&logo=windows&logoColor=white" alt="Download the Windows installer" height="42" /></a>
  <a href="https://github.com/mositho/filtertool-desktop/releases/latest"><img src="https://img.shields.io/github/v/release/mositho/filtertool-desktop?label=Latest%20version&style=for-the-badge" alt="Latest Filtertool version" height="42" /></a>
</p>

[Portable Windows](https://github.com/mositho/filtertool-desktop/releases/latest/download/Filtertool.exe) · [Linux AppImage](https://github.com/mositho/filtertool-desktop/releases/latest/download/Filtertool-x86_64.AppImage)

</div>

## Get started

Download the [Windows installer](https://github.com/mositho/filtertool-desktop/releases/latest/download/Filtertool-Setup.exe), or choose [portable Windows, Linux AppImage, and other release files](https://github.com/mositho/filtertool-desktop/releases/latest).

The Linux AppImage requires GTK 4 and WebKitGTK 6.0 from your distribution.

Your filters, settings and sounds are kept in Filtertool's application data folder. Back it up to keep a copy of your work.

### Windows setup

- **Filtertool-Setup.exe** installs Filtertool for your Windows account.
- **Filtertool.exe** runs without installation.

Requires 64-bit Windows 10 or 11 and Microsoft Edge WebView2. The installer installs WebView2 if needed; standalone users can get it from [Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/).

TTS sound generation requires a separate FFmpeg installation. Press `Windows + R` and run `winget install ffmpeg`, or download FFmpeg from [gyan.dev](https://www.gyan.dev/ffmpeg/builds/) and add `ffmpeg.exe` to your `PATH`. Restart Filtertool after installing or changing `PATH`.

On Windows, user data is stored in **`%APPDATA%\Filtertool`**.

## Quick guide

1. **Check your game folder.** Filtertool detects your PoE filter path. Use **Browse** to choose another folder.
2. **Create a filter** in the sidebar. Choose **Caster Template**, **Attack Template**, or a blank starting template, then enter your own filter name and click **Create**.
3. **Save** keeps your configuration locally. It does not update the in-game filter.
4. **Export** saves your configuration and writes the `.filter` file and sound pack to your game folder. **Export All** updates every saved filter. Generating TTS sounds needs FFmpeg and an internet connection.
5. **Select it in PoE.** Choose the exported filter in the game's item-filter settings. After changes, export again and reload it in-game.

| Want to…                       | Use                                                                    |
| :----------------------------- | :--------------------------------------------------------------------- |
| Bring filters from the web app | **Import → JSON files** or **Web App folder**                          |
| Share a filter                 | **Share**, then copy or save the compact JSON                          |
| Include your defaults/styles   | Enable the option in **Share**; imports keep them local to that filter |
| Change defaults or styles      | **Global Settings**                                                    |
| Save / save and export         | `Ctrl+S` / `Ctrl+Shift+S`                                              |
| Undo / redo                    | `Ctrl+Z` / `Ctrl+Shift+Z`                                              |

Import reads existing web JSON without changing the originals. Folder import also preserves JSON defaults/styles and referenced custom MP3s. Large shared filters can exceed Discord's message limit; send the JSON file instead.

> [!CAUTION]
> Keep the exported sound folder alongside the `.filter` file. Moving or deleting it will break the filter's sounds. Check your filters after exporting.

[Filter terminology](docs/GLOSSARY.md) · [Contributing](docs/CONTRIBUTING.md) · [Third-party notices](docs/THIRD_PARTY.md)
