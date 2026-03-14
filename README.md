# env-edit

A simple GUI editor for environment variables, built in **Go** with **Fyne**.

## Features

- View all environment variables from the current process
- Search by both key and value
- Create, edit, and delete variables, including renaming keys
- Reload variables from the current process
- Import variables from a `.env` file
- Export variables to a `.env` file

## Getting Started

> This project requires Go 1.22+.
> On Windows with Fyne, you also need `CGO_ENABLED=1` and a C compiler (`gcc`) available in `PATH`.

```bash
go mod tidy
go run .
```

## Windows: Common Startup Issues

If `go run .` fails with an error similar to:

```text
imports github.com/go-gl/gl/v2.1/gl: build constraints exclude all Go files
```

then Go is running with `CGO_ENABLED=0`. Fyne uses OpenGL bindings that require CGO on Windows.

Check your Go configuration:

```powershell
go env CGO_ENABLED
```

If the value is `0`, enable it:

```powershell
go env -w CGO_ENABLED=1
```

Then install a C compiler, for example `gcc` via MSYS2/MinGW, and make sure `gcc --version` works in your terminal.

Once both requirements are in place, you can start the app with:

```powershell
go run .
```

## Build `env-edit.exe` in Docker

There is now a Docker-based build path that cross-compiles the Windows binary with MinGW inside the container. That means you can build `env-edit.exe` without installing `gcc` locally.

Requirements:

- Docker Desktop or another working Docker installation
- the Docker engine must actually be running

Build the binary from PowerShell:

```powershell
.\scripts\build-windows-exe.ps1
```

This creates `env-edit.exe` in the project root. By default it is built as a Windows GUI app, so launching the `.exe` directly shows only the Fyne window and not an extra console window.

If you want a different output path:

```powershell
.\scripts\build-windows-exe.ps1 -OutputPath "dist\env-edit.exe"
```

If you explicitly want a console-attached build:

```powershell
.\scripts\build-windows-exe.ps1 -Console
```

Manual Docker build also works:

```powershell
docker build --build-arg GO_LDFLAGS="-H windowsgui" --target artifact --output type=local,dest=.\dist .
Copy-Item .\dist\env-edit.exe .\env-edit.exe
```

## Build for Windows

Build a Windows binary (`.exe`) from any platform:

```powershell
$env:CGO_ENABLED="1"
go build -ldflags="-H windowsgui" -o env-edit.exe .
```

Optional: for Windows ARM64:

```powershell
$env:CGO_ENABLED="1"
$env:GOARCH="arm64"
go build -o env-edit-arm64.exe .
```

## Notes

The app edits values in the program's memory and can import/export `.env` files.
Persistently setting global system environment variables in the operating system, for example via the Windows registry, is outside the scope of this version.
