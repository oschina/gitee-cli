#!/usr/bin/env node

const childProcess = require('child_process');

const BINARY_MAP = {
  darwin_x64:    { name: 'gitee-cli-darwin-amd64',   suffix: '' },
  darwin_arm64:  { name: 'gitee-cli-darwin-arm64',   suffix: '' },
  linux_x64:     { name: 'gitee-cli-linux-amd64',    suffix: '' },
  linux_arm64:   { name: 'gitee-cli-linux-arm64',    suffix: '' },
  win32_x64:     { name: 'gitee-cli-windows-amd64',  suffix: '.exe' },
  win32_arm64:   { name: 'gitee-cli-windows-arm64',  suffix: '.exe' },
};

const resolveBinaryPath = () => {
  try {
    const binary = BINARY_MAP[`${process.platform}_${process.arch}`];
    if (!binary) {
      console.error(`Unsupported platform/arch: ${process.platform}/${process.arch}`);
      process.exit(1);
    }
    return require.resolve(`@gitee/${binary.name}/bin/${binary.name}${binary.suffix}`);
  } catch (e) {
    console.error(
      `Could not resolve binary for ${process.platform}/${process.arch}. ` +
      'This likely means the platform-specific package was not installed. ' +
      'Try reinstalling @gitee/gitee-cli.'
    );
    process.exit(1);
  }
};

const result = childProcess.spawnSync(resolveBinaryPath(), process.argv.slice(2), {
  stdio: 'inherit',
});

if (result.error) {
  console.error(`Failed to execute gitee: ${result.error.message}`);
  process.exit(1);
}
if (result.signal) {
  process.kill(process.pid, result.signal);
}
process.exit(result.status === null ? 1 : result.status);
