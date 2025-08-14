# Cross-Compilation with zig-build

This project includes a cross-compilation script for better-sqlite3 using zig-build, which enables building native SQLite binaries for multiple platforms from a single development machine.

## Overview

The `scripts/build-sqlite.js` script uses the zig-build npm package to cross-compile better-sqlite3 for multiple platforms. This is particularly useful for:

- Creating binaries for deployment environments where compilation tools are not available
- Speeding up CI/CD pipelines by avoiding rebuild steps
- Supporting platforms where traditional cross-compilation is difficult

## Prerequisites

- Node.js 16+ with ES modules support
- Internet access (to download Zig toolchain and Node.js headers)
- npm package `zig-build` installed (automatically installed with `npm install`)
- better-sqlite3 package with source files available

## Usage

```bash
# Install dependencies (if not already done)
npm install

# Run cross-compilation
npm run build-sqlite
```

## Supported Platforms

The script builds binaries for the following platforms:

- **Windows x64** (`win32-x64`): x86_64-windows-gnu target
- **Linux x64** (`linux-x64`): x86_64-linux-gnu target with glibc 2.17 compatibility
- **Linux ARM64** (`linux-arm64`): aarch64-linux-gnu target with glibc 2.17 compatibility  
- **macOS x64** (`darwin-x64`): x86_64-macos target
- **macOS ARM64** (`darwin-arm64`): aarch64-macos target (Apple Silicon)

## Output

Cross-compiled binaries are placed in the `prebuilt/` directory:

```
prebuilt/
├── win32-x64/better_sqlite3.node
├── linux-x64/better_sqlite3.node
├── linux-arm64/better_sqlite3.node
├── darwin-x64/better_sqlite3.node
└── darwin-arm64/better_sqlite3.node
```

## Configuration

The build script includes optimized configurations for each platform:

- **SQLite Features**: Enables FTS3/4/5, JSON1, RTREE, and thread safety
- **C++ Standard**: Uses C++20 for modern language features
- **Linux Compatibility**: Targets glibc 2.17 for broad compatibility
- **Linking**: Includes platform-specific linker flags for optimal performance

## Troubleshooting

### "better-sqlite3 not found in node_modules"
Run `npm install` to ensure better-sqlite3 is installed.

### "Source file not found"
If better-sqlite3 was installed from prebuilt binaries, try:
```bash
npm rebuild better-sqlite3 --build-from-source
```

### Network/Download Issues
Ensure you have internet access. The first run downloads:
- Zig toolchain (~40MB)
- Node.js headers (~2MB)

These are cached locally for subsequent runs.

### Platform-Specific Build Failures
Some platforms may fail due to environment constraints. This is normal - the script will report which platforms succeeded.

## Technical Details

### zig-build Advantages

- **No System Compiler**: Uses Zig's bundled Clang, eliminating dependency on system tools
- **True Cross-Compilation**: Build for any target from any host platform
- **Reproducible Builds**: Deterministic output with cached toolchains
- **Modern C++**: Statically linked libc++ enables C++20 on any target

### SQLite Configuration

The build includes these SQLite compile-time options:
- `SQLITE_ENABLE_FTS3=1` - Full-text search version 3
- `SQLITE_ENABLE_FTS4=1` - Full-text search version 4  
- `SQLITE_ENABLE_FTS5=1` - Full-text search version 5
- `SQLITE_ENABLE_JSON1=1` - JSON SQL functions
- `SQLITE_ENABLE_RTREE=1` - R-tree spatial indexing
- `SQLITE_THREADSAFE=1` - Thread-safe operations

## Integration with CI/CD

Example GitHub Actions workflow:

```yaml
name: Cross-compile SQLite
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm install
      - run: npm run build-sqlite
      - uses: actions/upload-artifact@v3
        with:
          name: sqlite-binaries
          path: prebuilt/
```

This enables building once and using the binaries across multiple deployment targets.