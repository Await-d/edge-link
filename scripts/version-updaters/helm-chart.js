#!/usr/bin/env node
/**
 * scripts/version-updaters/helm-chart.js
 * Custom version updater for Helm Chart.yaml files
 * Used by standard-version to update both version and appVersion fields
 */

const fs = require('fs');
const yaml = require('js-yaml');

/**
 * standard-version updater interface
 * @param {string} contents - File contents
 * @param {string} version - New version to set
 * @returns {string} Updated file contents
 */
module.exports.readVersion = function (contents) {
  try {
    const chart = yaml.load(contents);
    return chart.version || '0.0.0';
  } catch (error) {
    console.error('Error reading Chart.yaml version:', error.message);
    return '0.0.0';
  }
};

module.exports.writeVersion = function (contents, version) {
  try {
    const chart = yaml.load(contents);

    // Update both version and appVersion
    chart.version = version;
    chart.appVersion = version;

    // Dump back to YAML with proper formatting
    return yaml.dump(chart, {
      indent: 2,
      lineWidth: -1, // No line wrapping
      noRefs: true,
      sortKeys: false // Preserve key order
    });
  } catch (error) {
    console.error('Error writing Chart.yaml version:', error.message);
    return contents;
  }
};
