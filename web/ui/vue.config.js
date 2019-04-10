module.exports = {
    publicPath: process.env.NODE_ENV === 'production' ? '/ui/' : '/',
    assetsDir: 'assets',
    devServer: {
        disableHostCheck: true
    }
}
