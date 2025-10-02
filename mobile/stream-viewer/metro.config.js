// metro.config.js
const path = require('path');
const { getDefaultConfig } = require('expo/metro-config');

/** @type {import('expo/metro-config').MetroConfig} */
const config = getDefaultConfig(__dirname);

// Some trees resolve the package "exports" weirdly; force CJS entry.
const structuredCloneCJS = path.resolve(
  __dirname,
  'node_modules/@ungap/structured-clone/cjs/index.js'
);

const defaultResolve = config.resolver.resolveRequest;
config.resolver.resolveRequest = (context, moduleName, platform) => {
  if (moduleName === '@ungap/structured-clone') {
    return { type: 'sourceFile', filePath: structuredCloneCJS };
  }
  return defaultResolve
    ? defaultResolve(context, moduleName, platform)
    : context.resolveRequest(context, moduleName, platform);
};

module.exports = config;
