const commonBuilder = require('./pulito/webpack.common.js');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const { resolve } = require('path');

module.exports = (env, argv) => {
  const config = commonBuilder(env, argv, __dirname);
  config.output.publicPath = '/dist/';
  config.resolve = config.resolve || {};
  // https://github.com/webpack/node-libs-browser/issues/26#issuecomment-267954095
  config.resolve.modules = [resolve(__dirname, 'node_modules'), 'node_modules'];

  if (argv.mode !== 'production') {
    config.plugins.push(
      new CopyWebpackPlugin([
        {
          from: resolve(__dirname, 'demo-page-assets'),
        },
      ]),
    );
  }

  return config;
};
