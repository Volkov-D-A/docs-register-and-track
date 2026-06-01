#!/usr/bin/env node

const cp = require("child_process");
const fs = require("fs");
const path = require("path");

const rootDir = path.resolve(__dirname, "..");
const frontendDir = path.join(rootDir, "frontend");
const evidenceDir = path.join(rootDir, "build", "release-evidence");
const reportPath = path.join(evidenceDir, "LICENSE_REPORT.md");

const disallowedPattern = /(^|[^A-Z])(AGPL|GPL|LGPL)([^A-Z]|$)/i;
const npmLicenseOverrides = {
  "node_modules/@antv/g2-extension-plot": {
    license: "MIT",
    reason:
      "package.json omits license, but package tarball includes LICENSE with MIT text.",
  },
};

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function run(command, args, options = {}) {
  const env = { ...process.env };
  if (command === "go") {
    env.GOFLAGS = `${env.GOFLAGS || ""} -mod=readonly`.trim();
  }

  return cp.execFileSync(command, args, {
    cwd: rootDir,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
    env,
    ...options,
  });
}

function normalizeLicense(raw) {
  return String(raw || "").trim();
}

function classifyLicenseText(text) {
  const sample = text
    .slice(0, 20000)
    .replace(/[“”]/g, '"')
    .replace(/[‘’]/g, "'")
    .toLowerCase();

  if (sample.includes("mozilla public license version 2.0")) return "MPL-2.0";
  if (sample.includes("apache license") && sample.includes("version 2.0")) {
    return "Apache-2.0";
  }
  if (
    sample.includes("permission is hereby granted, free of charge") &&
    sample.includes("the software is provided \"as is\"")
  ) {
    return "MIT";
  }
  if (
    sample.includes("redistribution and use in source and binary forms") &&
    sample.includes("neither the name")
  ) {
    return "BSD-3-Clause";
  }
  if (
    sample.includes("redistribution and use in source and binary forms") &&
    sample.includes("this software is provided by the copyright holders")
  ) {
    return "BSD-2-Clause";
  }
  if (
    sample.includes("permission to use, copy, modify, and/or distribute this software") &&
    sample.includes("the software is provided \"as is\"")
  ) {
    return "ISC";
  }
  if (sample.includes("## license") && sample.includes("mit license")) return "MIT";
  if (sample.includes("unlicense")) return "Unlicense";

  return "";
}

function findLicenseFile(moduleDir) {
  if (!moduleDir || !fs.existsSync(moduleDir)) return "";
  const names = fs.readdirSync(moduleDir);
  const match = names.find((name) => /^(license|licence|copying|notice)(\..*)?$/i.test(name));
  if (match) return path.join(moduleDir, match);

  const readme = names.find((name) => /^readme(\..*)?$/i.test(name));
  return readme ? path.join(moduleDir, readme) : "";
}

function collectNpmLicenses() {
  const lock = readJson(path.join(frontendDir, "package-lock.json"));
  const packages = Object.entries(lock.packages || {})
    .filter(([pkgPath]) => pkgPath)
    .map(([pkgPath, pkg]) => {
      const override = npmLicenseOverrides[pkgPath];
      const license = normalizeLicense(pkg.license || override?.license);
      return {
        ecosystem: "npm",
        name: pkg.name || pkgPath.replace(/^node_modules\//, ""),
        version: pkg.version || "",
        path: pkgPath,
        license,
        overrideReason: override?.reason || "",
      };
    });

  return packages;
}

function collectGoLicenses() {
  run("go", ["mod", "download", "all"]);
  const output = run("go", ["list", "-m", "-f", "{{.Path}}\t{{.Version}}\t{{.Dir}}", "all"]);
  return output
    .trim()
    .split(/\r?\n/)
    .filter(Boolean)
    .map((line) => {
      const [modulePath, version, moduleDir] = line.split("\t");
      const licenseFile = findLicenseFile(moduleDir);
      const license = moduleDir === rootDir
        ? "Project-owned"
        : licenseFile
        ? classifyLicenseText(fs.readFileSync(licenseFile, "utf8"))
        : "";
      return {
        ecosystem: "go",
        name: modulePath,
        version: version || "local",
        path: moduleDir || "",
        license,
        licenseFile,
      };
    });
}

function summarize(items) {
  const counts = new Map();
  for (const item of items) {
    const key = item.license || "UNKNOWN";
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return [...counts.entries()].sort(([a], [b]) => a.localeCompare(b));
}

function formatTable(items) {
  return [
    "| Ecosystem | Package | Version | License | Evidence |",
    "| --- | --- | --- | --- | --- |",
    ...items.map((item) => {
      const evidence = item.overrideReason || (item.licenseFile ? path.relative(rootDir, item.licenseFile) : "");
      return `| ${item.ecosystem} | \`${item.name}\` | \`${item.version}\` | ${item.license || "UNKNOWN"} | ${evidence || "-"} |`;
    }),
  ].join("\n");
}

function main() {
  const npmItems = collectNpmLicenses();
  const goItems = collectGoLicenses();
  const allItems = [...npmItems, ...goItems];

  const unknown = allItems.filter((item) => !item.license);
  const disallowed = allItems.filter((item) => disallowedPattern.test(item.license));

  fs.mkdirSync(evidenceDir, { recursive: true });
  const report = [
    "# License Report",
    "",
    "Generated by `node tools/license-report.js` from `frontend/package-lock.json` and `go list -m`.",
    "",
    "## Policy",
    "",
    "- Packages without a detected license are blocked unless listed in `npmLicenseOverrides` with evidence.",
    "- GPL, LGPL and AGPL family licenses are blocked for the release artifact unless a documented exception is approved.",
    "",
    "## Summary",
    "",
    `- npm packages checked: ${npmItems.length}`,
    `- Go modules checked: ${goItems.length}`,
    `- Unknown licenses: ${unknown.length}`,
    `- Disallowed licenses: ${disallowed.length}`,
    "",
    "## License Counts",
    "",
    "| License | Count |",
    "| --- | ---: |",
    ...summarize(allItems).map(([license, count]) => `| ${license} | ${count} |`),
    "",
    "## Inventory",
    "",
    formatTable(allItems),
    "",
  ].join("\n");

  fs.writeFileSync(reportPath, report);

  if (unknown.length || disallowed.length) {
    if (unknown.length) {
      console.error("Unknown licenses found:");
      for (const item of unknown) console.error(`- ${item.ecosystem}: ${item.name}@${item.version}`);
    }
    if (disallowed.length) {
      console.error("Disallowed licenses found:");
      for (const item of disallowed) {
        console.error(`- ${item.ecosystem}: ${item.name}@${item.version}: ${item.license}`);
      }
    }
    process.exit(1);
  }

  console.log(`License report written to ${path.relative(rootDir, reportPath)}`);
}

main();
