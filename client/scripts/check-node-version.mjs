const MIN_MAJOR = 20;
const MIN_MINOR = 19;
const MIN_MAJOR_ALT = 22;
const MIN_MINOR_ALT = 12;

const version = process.versions.node;
const [majorText, minorText] = version.split(".");
const major = Number(majorText);
const minor = Number(minorText);

const supported =
  (major === MIN_MAJOR && minor >= MIN_MINOR) ||
  (major === MIN_MAJOR_ALT && minor >= MIN_MINOR_ALT) ||
  major > MIN_MAJOR_ALT;

if (!supported) {
  console.error(
    [
      `Node.js ${version} is not supported.`,
      "Vite 8 requires Node.js 20.19+ or 22.12+.",
      "Upgrade Node.js, then run: cd client && npm install",
      "With nvm: nvm install && nvm use",
    ].join("\n"),
  );
  process.exit(1);
}
