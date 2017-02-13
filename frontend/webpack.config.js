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
      loader: 'babel',
      query: {
        presets: ['react', 'es2015', 'stage-1']
      }
    }]
  },
  resolve: {
    extensions: ['', '.js', '.jsx']
  },
  devServer: {
    historyApiFallback: true,
    contentBase: './',
    proxy: [
      {
        context: ['/file/**', '/folder*', '/list/**', '/tags*', '/account/**'],
        target: 'http://localhost:3000',
        secure: false
      }
    ]
  }
};
