# Tray icon assets

`tray.ico` is the icon shown in the Windows system tray. It is currently a copy of `build/windows/icon.ico` (the executable icon) so that the tray icon visually matches the application.

## Regenerating

If you update the executable icon at `build/windows/icon.ico`, refresh this copy:

```powershell
Copy-Item build\windows\icon.ico internal\tray\assets\tray.ico -Force
```

Tray icons render best with a multi-resolution `.ico` containing 16x16 and 32x32 variants. If a future change introduces a tray-specific source, regenerate that variant here and document the source-of-truth path in this README.
