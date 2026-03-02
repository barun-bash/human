/**
 * Generate platform-specific icons from the Human Studio logo SVG.
 *
 * Prerequisites: npm install -D sharp
 *
 * Usage: npx ts-node scripts/generate-icons.ts
 *
 * Creates:
 * - resources/icon.png (1024x1024 master)
 * - resources/icons/16x16.png
 * - resources/icons/32x32.png
 * - resources/icons/48x48.png
 * - resources/icons/128x128.png
 * - resources/icons/256x256.png
 * - resources/icons/512x512.png
 *
 * For .ico and .icns:
 *   macOS: iconutil -c icns resources/icon.iconset
 *   Windows: convert resources/icon.png -define icon:auto-resize=256,48,32,16 resources/icon.ico
 *   Or use: npx electron-icon-builder --input=resources/icon.png --output=resources/
 */

import { readFileSync, mkdirSync, existsSync } from 'fs'
import { join } from 'path'

async function main() {
  let sharp: any
  try {
    sharp = (await import('sharp')).default
  } catch {
    console.error('sharp is not installed. Run: npm install -D sharp')
    process.exit(1)
  }

  const svgPath = join(__dirname, '..', 'resources', 'icon.svg')
  const outDir = join(__dirname, '..', 'resources', 'icons')

  if (!existsSync(outDir)) {
    mkdirSync(outDir, { recursive: true })
  }

  const svgBuffer = readFileSync(svgPath)
  const sizes = [16, 32, 48, 128, 256, 512, 1024]

  for (const size of sizes) {
    const outPath = size === 1024
      ? join(__dirname, '..', 'resources', 'icon.png')
      : join(outDir, `${size}x${size}.png`)

    await sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(outPath)

    console.log(`  Created ${size}x${size} → ${outPath.split('resources/').pop()}`)
  }

  console.log('\nDone! For platform-specific formats:')
  console.log('  macOS:   iconutil -c icns resources/icon.iconset')
  console.log('  Windows: Use ImageMagick or electron-icon-builder')
  console.log('  Or:      npx electron-icon-builder --input=resources/icon.png --output=resources/')
}

main().catch(console.error)
