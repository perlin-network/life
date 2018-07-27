const fs = require("fs")

async function run() {
    let code = fs.readFileSync(process.argv[2])
    let module = await WebAssembly.compile(code)
    let instance = await WebAssembly.instantiate(module, {})
    let ret = instance.exports.app_main()
    console.log(ret)
}

run()
