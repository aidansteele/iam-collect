#!/usr/bin/env node
/**
 * Cross-compilation build script for better-sqlite3 using zig-build
 * This script compiles better-sqlite3 for multiple platforms using Zig's cross-compilation capabilities
 * 
 * Requirements:
 * - Internet access to download Zig toolchain and Node.js headers
 * - npm package 'zig-build' installed
 * - better-sqlite3 package installed in node_modules
 * 
 * Usage:
 *   npm run build-sqlite
 * 
 * Output:
 *   Cross-compiled binaries will be placed in the 'prebuilt' directory
 */

import { build } from 'zig-build'
import { readFileSync, copyFileSync, mkdirSync, existsSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const projectRoot = join(__dirname, '..')

// Find better-sqlite3 in node_modules
const betterSqlitePath = join(projectRoot, 'node_modules', 'better-sqlite3')

if (!existsSync(betterSqlitePath)) {
  console.error('❌ better-sqlite3 not found in node_modules. Please run npm install first.')
  process.exit(1)
}

// Output directory for cross-compiled binaries
const outputDir = join(projectRoot, 'prebuilt')
mkdirSync(outputDir, { recursive: true })

// Extract source files needed for compilation
const sources = [
  join(betterSqlitePath, 'src', 'better_sqlite3.cpp'),
  join(betterSqlitePath, 'deps', 'sqlite3', 'sqlite3.c'),
]

// Check if source files exist
for (const source of sources) {
  if (!existsSync(source)) {
    console.error(`❌ Source file not found: ${source}`)
    console.error('This may happen if better-sqlite3 was installed from prebuilt binaries.')
    console.error('Try running: npm rebuild better-sqlite3 --build-from-source')
    process.exit(1)
  }
}

// Common build configuration
const baseConfig = {
  sources,
  std: 'c++20',
  includes: [
    join(betterSqlitePath, 'src'),
    join(betterSqlitePath, 'deps', 'sqlite3'),
  ],
  defines: {
    SQLITE_ENABLE_FTS3: 1,
    SQLITE_ENABLE_FTS4: 1,
    SQLITE_ENABLE_FTS5: 1,
    SQLITE_ENABLE_JSON1: 1,
    SQLITE_ENABLE_RTREE: 1,
    SQLITE_THREADSAFE: 1,
    NAPI_VERSION: 6,
  },
}

// Platform-specific configurations
const platforms = {
  'win32-x64': {
    target: 'x86_64-windows-gnu',
    output: join(outputDir, 'win32-x64', 'better_sqlite3.node'),
    ...baseConfig,
  },
  'linux-x64': {
    target: 'x86_64-linux-gnu',
    output: join(outputDir, 'linux-x64', 'better_sqlite3.node'),
    glibc: '2.17', // For compatibility with older Linux distributions
    cflags: ['-Wl,-Bsymbolic', '-Wl,--exclude-libs,ALL'],
    ...baseConfig,
  },
  'linux-arm64': {
    target: 'aarch64-linux-gnu',
    output: join(outputDir, 'linux-arm64', 'better_sqlite3.node'),
    glibc: '2.17',
    cflags: ['-Wl,-Bsymbolic', '-Wl,--exclude-libs,ALL'],
    ...baseConfig,
  },
  'darwin-x64': {
    target: 'x86_64-macos',
    output: join(outputDir, 'darwin-x64', 'better_sqlite3.node'),
    ...baseConfig,
  },
  'darwin-arm64': {
    target: 'aarch64-macos',
    output: join(outputDir, 'darwin-arm64', 'better_sqlite3.node'),
    ...baseConfig,
  },
}

// Create output directories
for (const platform of Object.keys(platforms)) {
  const outputPath = platforms[platform].output
  mkdirSync(dirname(outputPath), { recursive: true })
}

async function main() {
  console.log('🚀 Starting cross-compilation of better-sqlite3...')
  console.log('📦 Source files found:')
  sources.forEach(source => console.log(`   - ${source}`))
  console.log('🎯 Target platforms:', Object.keys(platforms).join(', '))
  console.log('')
  
  try {
    await build(platforms)
    console.log('✅ Cross-compilation completed successfully!')
    console.log('📁 Prebuilt binaries available in:', outputDir)
    console.log('')
    
    // List generated files
    let successCount = 0
    for (const [platform, config] of Object.entries(platforms)) {
      if (existsSync(config.output)) {
        console.log(`✅ ${platform}: ${config.output}`)
        successCount++
      } else {
        console.log(`❌ ${platform}: Build failed`)
      }
    }
    
    console.log('')
    console.log(`🎉 Successfully built ${successCount}/${Object.keys(platforms).length} platforms`)
    
    if (successCount > 0) {
      console.log('')
      console.log('💡 These prebuilt binaries can be used to:')
      console.log('   - Deploy to environments where compilation is not available')
      console.log('   - Speed up CI/CD pipelines by avoiding rebuild')
      console.log('   - Support platforms where cross-compilation is difficult')
    }
    
  } catch (error) {
    console.error('❌ Cross-compilation failed:', error.message)
    console.error('')
    console.error('💡 Troubleshooting tips:')
    console.error('   - Ensure you have internet access for downloading Zig toolchain')
    console.error('   - Check that better-sqlite3 was installed with source files')
    console.error('   - Try running: npm rebuild better-sqlite3 --build-from-source')
    process.exit(1)
  }
}

main()