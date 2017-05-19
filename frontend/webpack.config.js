module.exports = {
  entry: [
    './src/index.js'
  ],
  output: {
    path: __dirname + "/dist",
    filename: "bundle.js"
  },
  module: {
    loaders: [
      { test: /\.css$/, loaders: ["style", "css"] },

      {
      exclude: /node_modules/,
      loader: 'babel-loader',
      query: {
        presets: ['react', 'es2015', 'stage-1']
      }
    }]
  },
  resolve: {
    extensions: ['.js', '.jsx']
  },
  devServer: {
    host: '0.0.0.0',
    disableHostCheck: true,
    historyApiFallback: true,
    hot: true,
    contentBase: './',
    proxy: [
      {
        context: ['/auth/**', '/account/**'],
        target: 'http://localhost:3000',
        secure: false
      }
    ]
  }
};
