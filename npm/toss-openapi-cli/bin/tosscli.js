#!/usr/bin/env node
"use strict";

const { execFileSync, spawnSync } = require("node:child_process");
const { globalPaths } = require("node:module");
const path = require("node:path");

const packages = {
  "darwin arm64": "@finetension/tosscli-darwin-arm64",
  "darwin x64": "@finetension/tosscli-darwin-x64",
  "linux arm64": "@finetension/tosscli-linux-arm64",
  "linux x64": "@finetension/tosscli-linux-x64",
  "win32 x64": "@finetension/tosscli-win32-x64",
};

const key = `${process.platform} ${process.arch}`;
const packageName = packages[key];

if (!packageName) {
  console.error(`tosscli: unsupported platform: ${key}`);
  process.exit(1);
}

const binaryName = process.platform === "win32" ? "tosscli.exe" : "tosscli";
let binaryPath;

try {
  binaryPath = require.resolve(`${packageName}/bin/${binaryName}`);
} catch (error) {
  const paths = [
    path.join(__dirname, "node_modules"),
    path.join(__dirname, ".."),
    path.join(__dirname, "..", ".."),
    path.join(path.dirname(process.argv[1]), "..", "lib", "node_modules"),
    ...globalPaths,
  ];

  try {
    const npmRoot = execFileSync("npm", ["root", "-g"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    if (npmRoot) {
      paths.unshift(npmRoot);
    }
  } catch (_) {
    // npm may not be available when this shim is invoked from a bundled context.
  }

  try {
    binaryPath = require.resolve(`${packageName}/bin/${binaryName}`, { paths });
  } catch (_) {
    console.error(`tosscli: platform package is missing: ${packageName}`);
    console.error("Reinstall with optional dependencies enabled:");
    console.error("  npm install -g toss-openapi-cli");
    process.exit(1);
  }
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  windowsHide: false,
});

if (result.error) {
  console.error(`tosscli: failed to execute ${binaryPath}: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
