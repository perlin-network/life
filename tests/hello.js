// The Module object: Our interface to the outside world. We import
// and export values on it. There are various ways Module can be used:
// 1. Not defined. We create it here
// 2. A function parameter, function(Module) { ..generated code.. }
// 3. pre-run appended it, var Module = {}; ..generated code..
// 4. External script tag defines var Module.
// We need to check if Module already exists (e.g. case 3 above).
// Substitution will be replaced with actual code on later stage of the build,
// this way Closure Compiler will not mangle it (e.g. case 4. above).
// Note that if you want to run closure, and also to use Module
// after the generated code, you will need to define   var Module = {};
// before the code. Then that object will be used in the code, and you
// can continue to use Module afterwards as well.
var Module = typeof Module !== 'undefined' ? Module : {};

// --pre-jses are emitted after the Module integration code, so that they can
// refer to Module (if they choose; they can also define Module)
// {{PRE_JSES}}

// Sometimes an existing Module object exists with properties
// meant to overwrite the default module functionality. Here
// we collect those properties and reapply _after_ we configure
// the current environment's defaults to avoid having to be so
// defensive during initialization.
var moduleOverrides = {};
var key;
for (key in Module) {
    if (Module.hasOwnProperty(key)) {
        moduleOverrides[key] = Module[key];
    }
}

Module['arguments'] = [];
Module['thisProgram'] = './this.program';
Module['quit'] = function (status, toThrow) {
    throw toThrow;
};
Module['preRun'] = [];
Module['postRun'] = [];

// The environment setup code below is customized to use Module.
// *** Environment setup code ***

var ENVIRONMENT_IS_WEB = false;
var ENVIRONMENT_IS_WORKER = false;
var ENVIRONMENT_IS_NODE = false;
var ENVIRONMENT_IS_SHELL = false;
ENVIRONMENT_IS_WEB = typeof window === 'object';
ENVIRONMENT_IS_WORKER = typeof importScripts === 'function';
ENVIRONMENT_IS_NODE = typeof process === 'object' && typeof require === 'function' && !ENVIRONMENT_IS_WEB && !ENVIRONMENT_IS_WORKER;
ENVIRONMENT_IS_SHELL = !ENVIRONMENT_IS_WEB && !ENVIRONMENT_IS_NODE && !ENVIRONMENT_IS_WORKER;

if (Module['ENVIRONMENT']) {
    throw new Error('Module.ENVIRONMENT has been deprecated. To force the environment, use the ENVIRONMENT compile-time option (for example, -s ENVIRONMENT=web or -s ENVIRONMENT=node)');
}

// Three configurations we can be running in:
// 1) We could be the application main() thread running in the main JS UI thread. (ENVIRONMENT_IS_WORKER == false and ENVIRONMENT_IS_PTHREAD == false)
// 2) We could be the application main() thread proxied to worker. (with Emscripten -s PROXY_TO_WORKER=1) (ENVIRONMENT_IS_WORKER == true, ENVIRONMENT_IS_PTHREAD == false)
// 3) We could be an application pthread running in a worker. (ENVIRONMENT_IS_WORKER == true and ENVIRONMENT_IS_PTHREAD == true)

assert(typeof Module['memoryInitializerPrefixURL'] === 'undefined', 'Module.memoryInitializerPrefixURL option was removed, use Module.locateFile instead');
assert(typeof Module['pthreadMainPrefixURL'] === 'undefined', 'Module.pthreadMainPrefixURL option was removed, use Module.locateFile instead');
assert(typeof Module['cdInitializerPrefixURL'] === 'undefined', 'Module.cdInitializerPrefixURL option was removed, use Module.locateFile instead');
assert(typeof Module['filePackagePrefixURL'] === 'undefined', 'Module.filePackagePrefixURL option was removed, use Module.locateFile instead');

// `/` should be present at the end if `scriptDirectory` is not empty
var scriptDirectory = '';

function locateFile(path) {
    if (Module['locateFile']) {
        return Module['locateFile'](path, scriptDirectory);
    } else {
        return scriptDirectory + path;
    }
}

if (ENVIRONMENT_IS_NODE) {
    scriptDirectory = __dirname + '/';

    // Expose functionality in the same simple way that the shells work
    // Note that we pollute the global namespace here, otherwise we break in node
    var nodeFS;
    var nodePath;

    Module['read'] = function shell_read(filename, binary) {
        var ret;
        if (!nodeFS) nodeFS = require('fs');
        if (!nodePath) nodePath = require('path');
        filename = nodePath['normalize'](filename);
        ret = nodeFS['readFileSync'](filename);
        return binary ? ret : ret.toString();
    };

    Module['readBinary'] = function readBinary(filename) {
        var ret = Module['read'](filename, true);
        if (!ret.buffer) {
            ret = new Uint8Array(ret);
        }
        assert(ret.buffer);
        return ret;
    };

    if (process['argv'].length > 1) {
        Module['thisProgram'] = process['argv'][1].replace(/\\/g, '/');
    }

    Module['arguments'] = process['argv'].slice(2);

    if (typeof module !== 'undefined') {
        module['exports'] = Module;
    }

    process['on']('uncaughtException', function (ex) {
        // suppress ExitStatus exceptions from showing an error
        if (!(ex instanceof ExitStatus)) {
            throw ex;
        }
    });
    // Currently node will swallow unhandled rejections, but this behavior is
    // deprecated, and in the future it will exit with error status.
    process['on']('unhandledRejection', function (reason, p) {
        err('node.js exiting due to unhandled promise rejection');
        process['exit'](1);
    });

    Module['quit'] = function (status) {
        process['exit'](status);
    };

    Module['inspect'] = function () {
        return '[Emscripten Module object]';
    };
} else if (ENVIRONMENT_IS_SHELL) {


    if (typeof read != 'undefined') {
        Module['read'] = function shell_read(f) {
            return read(f);
        };
    }

    Module['readBinary'] = function readBinary(f) {
        var data;
        if (typeof readbuffer === 'function') {
            return new Uint8Array(readbuffer(f));
        }
        data = read(f, 'binary');
        assert(typeof data === 'object');
        return data;
    };

    if (typeof scriptArgs != 'undefined') {
        Module['arguments'] = scriptArgs;
    } else if (typeof arguments != 'undefined') {
        Module['arguments'] = arguments;
    }

    if (typeof quit === 'function') {
        Module['quit'] = function (status) {
            quit(status);
        }
    }
} else if (ENVIRONMENT_IS_WEB || ENVIRONMENT_IS_WORKER) {
    if (ENVIRONMENT_IS_WEB) {
        var currentScript = document.currentScript;
        if (currentScript.src.indexOf('blob:') !== 0) {
            scriptDirectory = currentScript.src.split('/').slice(0, -1).join('/') + '/';
        }
    } else if (ENVIRONMENT_IS_WORKER) {
        scriptDirectory = self.location.href.split('/').slice(0, -1).join('/') + '/';
    }


    Module['read'] = function shell_read(url) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', url, false);
        xhr.send(null);
        return xhr.responseText;
    };

    if (ENVIRONMENT_IS_WORKER) {
        Module['readBinary'] = function readBinary(url) {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', url, false);
            xhr.responseType = 'arraybuffer';
            xhr.send(null);
            return new Uint8Array(xhr.response);
        };
    }

    Module['readAsync'] = function readAsync(url, onload, onerror) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', url, true);
        xhr.responseType = 'arraybuffer';
        xhr.onload = function xhr_onload() {
            if (xhr.status == 200 || (xhr.status == 0 && xhr.response)) { // file URLs can return 0
                onload(xhr.response);
                return;
            }
            onerror();
        };
        xhr.onerror = onerror;
        xhr.send(null);
    };

    Module['setWindowTitle'] = function (title) {
        document.title = title
    };
} else {
    throw new Error('environment detection error');
}

// Set up the out() and err() hooks, which are how we can print to stdout or
// stderr, respectively.
// If the user provided Module.print or printErr, use that. Otherwise,
// console.log is checked first, as 'print' on the web will open a print dialogue
// printErr is preferable to console.warn (works better in shells)
// bind(console) is necessary to fix IE/Edge closed dev tools panel behavior.
var out = Module['print'] || (typeof console !== 'undefined' ? console.log.bind(console) : (typeof print !== 'undefined' ? print : null));
var err = Module['printErr'] || (typeof printErr !== 'undefined' ? printErr : ((typeof console !== 'undefined' && console.warn.bind(console)) || out));

// *** Environment setup code ***

// Merge back in the overrides
for (key in moduleOverrides) {
    if (moduleOverrides.hasOwnProperty(key)) {
        Module[key] = moduleOverrides[key];
    }
}
// Free the object hierarchy contained in the overrides, this lets the GC
// reclaim data used e.g. in memoryInitializerRequest, which is a large typed array.
moduleOverrides = undefined;


// {{PREAMBLE_ADDITIONS}}

var STACK_ALIGN = 16;

// stack management, and other functionality that is provided by the compiled code,
// should not be used before it is ready
stackSave = stackRestore = stackAlloc = setTempRet0 = getTempRet0 = function () {
    abort('cannot use the stack before compiled code is ready to run, and has provided stack access');
};

function staticAlloc(size) {
    assert(!staticSealed);
    var ret = STATICTOP;
    STATICTOP = (STATICTOP + size + 15) & -16;
    assert(STATICTOP < TOTAL_MEMORY, 'not enough memory for static allocation - increase TOTAL_MEMORY');
    return ret;
}

function dynamicAlloc(size) {
    assert(DYNAMICTOP_PTR);
    var ret = HEAP32[DYNAMICTOP_PTR >> 2];
    var end = (ret + size + 15) & -16;
    HEAP32[DYNAMICTOP_PTR >> 2] = end;
    if (end >= TOTAL_MEMORY) {
        var success = enlargeMemory();
        if (!success) {
            HEAP32[DYNAMICTOP_PTR >> 2] = ret;
            return 0;
        }
    }
    return ret;
}

function alignMemory(size, factor) {
    if (!factor) factor = STACK_ALIGN; // stack alignment (16-byte) by default
    var ret = size = Math.ceil(size / factor) * factor;
    return ret;
}

function getNativeTypeSize(type) {
    switch (type) {
        case 'i1':
        case 'i8':
            return 1;
        case 'i16':
            return 2;
        case 'i32':
            return 4;
        case 'i64':
            return 8;
        case 'float':
            return 4;
        case 'double':
            return 8;
        default: {
            if (type[type.length - 1] === '*') {
                return 4; // A pointer
            } else if (type[0] === 'i') {
                var bits = parseInt(type.substr(1));
                assert(bits % 8 === 0);
                return bits / 8;
            } else {
                return 0;
            }
        }
    }
}

function warnOnce(text) {
    if (!warnOnce.shown) warnOnce.shown = {};
    if (!warnOnce.shown[text]) {
        warnOnce.shown[text] = 1;
        err(text);
    }
}

var asm2wasmImports = { // special asm2wasm imports
    "f64-rem": function (x, y) {
        return x % y;
    },
    "debugger": function () {
        debugger;
    }
};


var jsCallStartIndex = 1;
var functionPointers = new Array(0);

// 'sig' parameter is only used on LLVM wasm backend
function addFunction(func, sig) {
    if (typeof sig === 'undefined') {
        err('warning: addFunction(): You should provide a wasm function signature string as a second argument. This is not necessary for asm.js and asm2wasm, but is required for the LLVM wasm backend, so it is recommended for full portability.');
    }
    var base = 0;
    for (var i = base; i < base + 0; i++) {
        if (!functionPointers[i]) {
            functionPointers[i] = func;
            return jsCallStartIndex + i;
        }
    }
    throw 'Finished up all reserved function pointers. Use a higher value for RESERVED_FUNCTION_POINTERS.';
}

function removeFunction(index) {
    functionPointers[index - jsCallStartIndex] = null;
}

var funcWrappers = {};

function getFuncWrapper(func, sig) {
    if (!func) return; // on null pointer, return undefined
    assert(sig);
    if (!funcWrappers[sig]) {
        funcWrappers[sig] = {};
    }
    var sigCache = funcWrappers[sig];
    if (!sigCache[func]) {
        // optimize away arguments usage in common cases
        if (sig.length === 1) {
            sigCache[func] = function dynCall_wrapper() {
                return dynCall(sig, func);
            };
        } else if (sig.length === 2) {
            sigCache[func] = function dynCall_wrapper(arg) {
                return dynCall(sig, func, [arg]);
            };
        } else {
            // general case
            sigCache[func] = function dynCall_wrapper() {
                return dynCall(sig, func, Array.prototype.slice.call(arguments));
            };
        }
    }
    return sigCache[func];
}


function makeBigInt(low, high, unsigned) {
    return unsigned ? ((+((low >>> 0))) + ((+((high >>> 0))) * 4294967296.0)) : ((+((low >>> 0))) + ((+((high | 0))) * 4294967296.0));
}

function dynCall(sig, ptr, args) {
    if (args && args.length) {
        assert(args.length == sig.length - 1);
        assert(('dynCall_' + sig) in Module, 'bad function pointer type - no table for sig \'' + sig + '\'');
        return Module['dynCall_' + sig].apply(null, [ptr].concat(args));
    } else {
        assert(sig.length == 1);
        assert(('dynCall_' + sig) in Module, 'bad function pointer type - no table for sig \'' + sig + '\'');
        return Module['dynCall_' + sig].call(null, ptr);
    }
}


function getCompilerSetting(name) {
    throw 'You must build with -s RETAIN_COMPILER_SETTINGS=1 for getCompilerSetting or emscripten_get_compiler_setting to work';
}

var Runtime = {
    // FIXME backwards compatibility layer for ports. Support some Runtime.*
    //       for now, fix it there, then remove it from here. That way we
    //       can minimize any period of breakage.
    dynCall: dynCall, // for SDL2 port
    // helpful errors
    getTempRet0: function () {
        abort('getTempRet0() is now a top-level function, after removing the Runtime object. Remove "Runtime."')
    },
    staticAlloc: function () {
        abort('staticAlloc() is now a top-level function, after removing the Runtime object. Remove "Runtime."')
    },
    stackAlloc: function () {
        abort('stackAlloc() is now a top-level function, after removing the Runtime object. Remove "Runtime."')
    },
};

// The address globals begin at. Very low in memory, for code size and optimization opportunities.
// Above 0 is static memory, starting with globals.
// Then the stack.
// Then 'dynamic' memory for sbrk.
var GLOBAL_BASE = 1024;


// === Preamble library stuff ===

// Documentation for the public APIs defined in this file must be updated in:
//    site/source/docs/api_reference/preamble.js.rst
// A prebuilt local version of the documentation is available at:
//    site/build/text/docs/api_reference/preamble.js.txt
// You can also build docs locally as HTML or other formats in site/
// An online HTML version (which may be of a different version of Emscripten)
//    is up at http://kripken.github.io/emscripten-site/docs/api_reference/preamble.js.html


//========================================
// Runtime essentials
//========================================

var ABORT = 0; // whether we are quitting the application. no code should run after this. set in exit() and abort()
var EXITSTATUS = 0;

/** @type {function(*, string=)} */
function assert(condition, text) {
    if (!condition) {
        abort('Assertion failed: ' + text);
    }
}

var globalScope = this;

// Returns the C function with a specified identifier (for C++, you need to do manual name mangling)
function getCFunc(ident) {
    var func = Module['_' + ident]; // closure exported function
    assert(func, 'Cannot call unknown function ' + ident + ', make sure it is exported');
    return func;
}

var JSfuncs = {
    // Helpers for cwrap -- it can't refer to Runtime directly because it might
    // be renamed by closure, instead it calls JSfuncs['stackSave'].body to find
    // out what the minified function name is.
    'stackSave': function () {
        stackSave()
    },
    'stackRestore': function () {
        stackRestore()
    },
    // type conversion from js to c
    'arrayToC': function (arr) {
        var ret = stackAlloc(arr.length);
        writeArrayToMemory(arr, ret);
        return ret;
    },
    'stringToC': function (str) {
        var ret = 0;
        if (str !== null && str !== undefined && str !== 0) { // null string
            // at most 4 bytes per UTF-8 code point, +1 for the trailing '\0'
            var len = (str.length << 2) + 1;
            ret = stackAlloc(len);
            stringToUTF8(str, ret, len);
        }
        return ret;
    }
};

// For fast lookup of conversion functions
var toC = {
    'string': JSfuncs['stringToC'], 'array': JSfuncs['arrayToC']
};


// C calling interface.
function ccall(ident, returnType, argTypes, args, opts) {
    function convertReturnValue(ret) {
        if (returnType === 'string') return Pointer_stringify(ret);
        if (returnType === 'boolean') return Boolean(ret);
        return ret;
    }

    var func = getCFunc(ident);
    var cArgs = [];
    var stack = 0;
    assert(returnType !== 'array', 'Return type should not be "array".');
    if (args) {
        for (var i = 0; i < args.length; i++) {
            var converter = toC[argTypes[i]];
            if (converter) {
                if (stack === 0) stack = stackSave();
                cArgs[i] = converter(args[i]);
            } else {
                cArgs[i] = args[i];
            }
        }
    }
    var ret = func.apply(null, cArgs);
    ret = convertReturnValue(ret);
    if (stack !== 0) stackRestore(stack);
    return ret;
}

function cwrap(ident, returnType, argTypes, opts) {
    return function () {
        return ccall(ident, returnType, argTypes, arguments, opts);
    }
}

/** @type {function(number, number, string, boolean=)} */
function setValue(ptr, value, type, noSafe) {
    type = type || 'i8';
    if (type.charAt(type.length - 1) === '*') type = 'i32'; // pointers are 32-bit
    switch (type) {
        case 'i1':
            HEAP8[((ptr) >> 0)] = value;
            break;
        case 'i8':
            HEAP8[((ptr) >> 0)] = value;
            break;
        case 'i16':
            HEAP16[((ptr) >> 1)] = value;
            break;
        case 'i32':
            HEAP32[((ptr) >> 2)] = value;
            break;
        case 'i64':
            (tempI64 = [value >>> 0, (tempDouble = value, (+(Math_abs(tempDouble))) >= 1.0 ? (tempDouble > 0.0 ? ((Math_min((+(Math_floor((tempDouble) / 4294967296.0))), 4294967295.0)) | 0) >>> 0 : (~~((+(Math_ceil((tempDouble - +(((~~(tempDouble))) >>> 0)) / 4294967296.0))))) >>> 0) : 0)], HEAP32[((ptr) >> 2)] = tempI64[0], HEAP32[(((ptr) + (4)) >> 2)] = tempI64[1]);
            break;
        case 'float':
            HEAPF32[((ptr) >> 2)] = value;
            break;
        case 'double':
            HEAPF64[((ptr) >> 3)] = value;
            break;
        default:
            abort('invalid type for setValue: ' + type);
    }
}

/** @type {function(number, string, boolean=)} */
function getValue(ptr, type, noSafe) {
    type = type || 'i8';
    if (type.charAt(type.length - 1) === '*') type = 'i32'; // pointers are 32-bit
    switch (type) {
        case 'i1':
            return HEAP8[((ptr) >> 0)];
        case 'i8':
            return HEAP8[((ptr) >> 0)];
        case 'i16':
            return HEAP16[((ptr) >> 1)];
        case 'i32':
            return HEAP32[((ptr) >> 2)];
        case 'i64':
            return HEAP32[((ptr) >> 2)];
        case 'float':
            return HEAPF32[((ptr) >> 2)];
        case 'double':
            return HEAPF64[((ptr) >> 3)];
        default:
            abort('invalid type for getValue: ' + type);
    }
    return null;
}

var ALLOC_NORMAL = 0; // Tries to use _malloc()
var ALLOC_STACK = 1; // Lives for the duration of the current function call
var ALLOC_STATIC = 2; // Cannot be freed
var ALLOC_DYNAMIC = 3; // Cannot be freed except through sbrk
var ALLOC_NONE = 4; // Do not allocate

// allocate(): This is for internal use. You can use it yourself as well, but the interface
//             is a little tricky (see docs right below). The reason is that it is optimized
//             for multiple syntaxes to save space in generated code. So you should
//             normally not use allocate(), and instead allocate memory using _malloc(),
//             initialize it with setValue(), and so forth.
// @slab: An array of data, or a number. If a number, then the size of the block to allocate,
//        in *bytes* (note that this is sometimes confusing: the next parameter does not
//        affect this!)
// @types: Either an array of types, one for each byte (or 0 if no type at that position),
//         or a single type which is used for the entire block. This only matters if there
//         is initial data - if @slab is a number, then this does not matter at all and is
//         ignored.
// @allocator: How to allocate memory, see ALLOC_*
/** @type {function((TypedArray|Array<number>|number), string, number, number=)} */
function allocate(slab, types, allocator, ptr) {
    var zeroinit, size;
    if (typeof slab === 'number') {
        zeroinit = true;
        size = slab;
    } else {
        zeroinit = false;
        size = slab.length;
    }

    var singleType = typeof types === 'string' ? types : null;

    var ret;
    if (allocator == ALLOC_NONE) {
        ret = ptr;
    } else {
        ret = [typeof _malloc === 'function' ? _malloc : staticAlloc, stackAlloc, staticAlloc, dynamicAlloc][allocator === undefined ? ALLOC_STATIC : allocator](Math.max(size, singleType ? 1 : types.length));
    }

    if (zeroinit) {
        var stop;
        ptr = ret;
        assert((ret & 3) == 0);
        stop = ret + (size & ~3);
        for (; ptr < stop; ptr += 4) {
            HEAP32[((ptr) >> 2)] = 0;
        }
        stop = ret + size;
        while (ptr < stop) {
            HEAP8[((ptr++) >> 0)] = 0;
        }
        return ret;
    }

    if (singleType === 'i8') {
        if (slab.subarray || slab.slice) {
            HEAPU8.set(/** @type {!Uint8Array} */ (slab), ret);
        } else {
            HEAPU8.set(new Uint8Array(slab), ret);
        }
        return ret;
    }

    var i = 0, type, typeSize, previousType;
    while (i < size) {
        var curr = slab[i];

        type = singleType || types[i];
        if (type === 0) {
            i++;
            continue;
        }
        assert(type, 'Must know what type to store in allocate!');

        if (type == 'i64') type = 'i32'; // special case: we have one i32 here, and one i32 later

        setValue(ret + i, curr, type);

        // no need to look up size unless type changes, so cache it
        if (previousType !== type) {
            typeSize = getNativeTypeSize(type);
            previousType = type;
        }
        i += typeSize;
    }

    return ret;
}

// Allocate memory during any stage of startup - static memory early on, dynamic memory later, malloc when ready
function getMemory(size) {
    if (!staticSealed) return staticAlloc(size);
    if (!runtimeInitialized) return dynamicAlloc(size);
    return _malloc(size);
}

/** @type {function(number, number=)} */
function Pointer_stringify(ptr, length) {
    if (length === 0 || !ptr) return '';
    // Find the length, and check for UTF while doing so
    var hasUtf = 0;
    var t;
    var i = 0;
    while (1) {
        assert(ptr + i < TOTAL_MEMORY);
        t = HEAPU8[(((ptr) + (i)) >> 0)];
        hasUtf |= t;
        if (t == 0 && !length) break;
        i++;
        if (length && i == length) break;
    }
    if (!length) length = i;

    var ret = '';

    if (hasUtf < 128) {
        var MAX_CHUNK = 1024; // split up into chunks, because .apply on a huge string can overflow the stack
        var curr;
        while (length > 0) {
            curr = String.fromCharCode.apply(String, HEAPU8.subarray(ptr, ptr + Math.min(length, MAX_CHUNK)));
            ret = ret ? ret + curr : curr;
            ptr += MAX_CHUNK;
            length -= MAX_CHUNK;
        }
        return ret;
    }
    return UTF8ToString(ptr);
}

// Given a pointer 'ptr' to a null-terminated ASCII-encoded string in the emscripten HEAP, returns
// a copy of that string as a Javascript String object.

function AsciiToString(ptr) {
    var str = '';
    while (1) {
        var ch = HEAP8[((ptr++) >> 0)];
        if (!ch) return str;
        str += String.fromCharCode(ch);
    }
}

// Copies the given Javascript String object 'str' to the emscripten HEAP at address 'outPtr',
// null-terminated and encoded in ASCII form. The copy will require at most str.length+1 bytes of space in the HEAP.

function stringToAscii(str, outPtr) {
    return writeAsciiToMemory(str, outPtr, false);
}

// Given a pointer 'ptr' to a null-terminated UTF8-encoded string in the given array that contains uint8 values, returns
// a copy of that string as a Javascript String object.

var UTF8Decoder = typeof TextDecoder !== 'undefined' ? new TextDecoder('utf8') : undefined;

function UTF8ArrayToString(u8Array, idx) {
    var endPtr = idx;
    // TextDecoder needs to know the byte length in advance, it doesn't stop on null terminator by itself.
    // Also, use the length info to avoid running tiny strings through TextDecoder, since .subarray() allocates garbage.
    while (u8Array[endPtr]) ++endPtr;

    if (endPtr - idx > 16 && u8Array.subarray && UTF8Decoder) {
        return UTF8Decoder.decode(u8Array.subarray(idx, endPtr));
    } else {
        var u0, u1, u2, u3, u4, u5;

        var str = '';
        while (1) {
            // For UTF8 byte structure, see http://en.wikipedia.org/wiki/UTF-8#Description and https://www.ietf.org/rfc/rfc2279.txt and https://tools.ietf.org/html/rfc3629
            u0 = u8Array[idx++];
            if (!u0) return str;
            if (!(u0 & 0x80)) {
                str += String.fromCharCode(u0);
                continue;
            }
            u1 = u8Array[idx++] & 63;
            if ((u0 & 0xE0) == 0xC0) {
                str += String.fromCharCode(((u0 & 31) << 6) | u1);
                continue;
            }
            u2 = u8Array[idx++] & 63;
            if ((u0 & 0xF0) == 0xE0) {
                u0 = ((u0 & 15) << 12) | (u1 << 6) | u2;
            } else {
                u3 = u8Array[idx++] & 63;
                if ((u0 & 0xF8) == 0xF0) {
                    u0 = ((u0 & 7) << 18) | (u1 << 12) | (u2 << 6) | u3;
                } else {
                    u4 = u8Array[idx++] & 63;
                    if ((u0 & 0xFC) == 0xF8) {
                        u0 = ((u0 & 3) << 24) | (u1 << 18) | (u2 << 12) | (u3 << 6) | u4;
                    } else {
                        u5 = u8Array[idx++] & 63;
                        u0 = ((u0 & 1) << 30) | (u1 << 24) | (u2 << 18) | (u3 << 12) | (u4 << 6) | u5;
                    }
                }
            }
            if (u0 < 0x10000) {
                str += String.fromCharCode(u0);
            } else {
                var ch = u0 - 0x10000;
                str += String.fromCharCode(0xD800 | (ch >> 10), 0xDC00 | (ch & 0x3FF));
            }
        }
    }
}

// Given a pointer 'ptr' to a null-terminated UTF8-encoded string in the emscripten HEAP, returns
// a copy of that string as a Javascript String object.

function UTF8ToString(ptr) {
    return UTF8ArrayToString(HEAPU8, ptr);
}

// Copies the given Javascript String object 'str' to the given byte array at address 'outIdx',
// encoded in UTF8 form and null-terminated. The copy will require at most str.length*4+1 bytes of space in the HEAP.
// Use the function lengthBytesUTF8 to compute the exact number of bytes (excluding null terminator) that this function will write.
// Parameters:
//   str: the Javascript string to copy.
//   outU8Array: the array to copy to. Each index in this array is assumed to be one 8-byte element.
//   outIdx: The starting offset in the array to begin the copying.
//   maxBytesToWrite: The maximum number of bytes this function can write to the array. This count should include the null
//                    terminator, i.e. if maxBytesToWrite=1, only the null terminator will be written and nothing else.
//                    maxBytesToWrite=0 does not write any bytes to the output, not even the null terminator.
// Returns the number of bytes written, EXCLUDING the null terminator.

function stringToUTF8Array(str, outU8Array, outIdx, maxBytesToWrite) {
    if (!(maxBytesToWrite > 0)) // Parameter maxBytesToWrite is not optional. Negative values, 0, null, undefined and false each don't write out any bytes.
        return 0;

    var startIdx = outIdx;
    var endIdx = outIdx + maxBytesToWrite - 1; // -1 for string null terminator.
    for (var i = 0; i < str.length; ++i) {
        // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code unit, not a Unicode code point of the character! So decode UTF16->UTF32->UTF8.
        // See http://unicode.org/faq/utf_bom.html#utf16-3
        // For UTF8 byte structure, see http://en.wikipedia.org/wiki/UTF-8#Description and https://www.ietf.org/rfc/rfc2279.txt and https://tools.ietf.org/html/rfc3629
        var u = str.charCodeAt(i); // possibly a lead surrogate
        if (u >= 0xD800 && u <= 0xDFFF) u = 0x10000 + ((u & 0x3FF) << 10) | (str.charCodeAt(++i) & 0x3FF);
        if (u <= 0x7F) {
            if (outIdx >= endIdx) break;
            outU8Array[outIdx++] = u;
        } else if (u <= 0x7FF) {
            if (outIdx + 1 >= endIdx) break;
            outU8Array[outIdx++] = 0xC0 | (u >> 6);
            outU8Array[outIdx++] = 0x80 | (u & 63);
        } else if (u <= 0xFFFF) {
            if (outIdx + 2 >= endIdx) break;
            outU8Array[outIdx++] = 0xE0 | (u >> 12);
            outU8Array[outIdx++] = 0x80 | ((u >> 6) & 63);
            outU8Array[outIdx++] = 0x80 | (u & 63);
        } else if (u <= 0x1FFFFF) {
            if (outIdx + 3 >= endIdx) break;
            outU8Array[outIdx++] = 0xF0 | (u >> 18);
            outU8Array[outIdx++] = 0x80 | ((u >> 12) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 6) & 63);
            outU8Array[outIdx++] = 0x80 | (u & 63);
        } else if (u <= 0x3FFFFFF) {
            if (outIdx + 4 >= endIdx) break;
            outU8Array[outIdx++] = 0xF8 | (u >> 24);
            outU8Array[outIdx++] = 0x80 | ((u >> 18) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 12) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 6) & 63);
            outU8Array[outIdx++] = 0x80 | (u & 63);
        } else {
            if (outIdx + 5 >= endIdx) break;
            outU8Array[outIdx++] = 0xFC | (u >> 30);
            outU8Array[outIdx++] = 0x80 | ((u >> 24) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 18) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 12) & 63);
            outU8Array[outIdx++] = 0x80 | ((u >> 6) & 63);
            outU8Array[outIdx++] = 0x80 | (u & 63);
        }
    }
    // Null-terminate the pointer to the buffer.
    outU8Array[outIdx] = 0;
    return outIdx - startIdx;
}

// Copies the given Javascript String object 'str' to the emscripten HEAP at address 'outPtr',
// null-terminated and encoded in UTF8 form. The copy will require at most str.length*4+1 bytes of space in the HEAP.
// Use the function lengthBytesUTF8 to compute the exact number of bytes (excluding null terminator) that this function will write.
// Returns the number of bytes written, EXCLUDING the null terminator.

function stringToUTF8(str, outPtr, maxBytesToWrite) {
    assert(typeof maxBytesToWrite == 'number', 'stringToUTF8(str, outPtr, maxBytesToWrite) is missing the third parameter that specifies the length of the output buffer!');
    return stringToUTF8Array(str, HEAPU8, outPtr, maxBytesToWrite);
}

// Returns the number of bytes the given Javascript string takes if encoded as a UTF8 byte array, EXCLUDING the null terminator byte.

function lengthBytesUTF8(str) {
    var len = 0;
    for (var i = 0; i < str.length; ++i) {
        // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code unit, not a Unicode code point of the character! So decode UTF16->UTF32->UTF8.
        // See http://unicode.org/faq/utf_bom.html#utf16-3
        var u = str.charCodeAt(i); // possibly a lead surrogate
        if (u >= 0xD800 && u <= 0xDFFF) u = 0x10000 + ((u & 0x3FF) << 10) | (str.charCodeAt(++i) & 0x3FF);
        if (u <= 0x7F) {
            ++len;
        } else if (u <= 0x7FF) {
            len += 2;
        } else if (u <= 0xFFFF) {
            len += 3;
        } else if (u <= 0x1FFFFF) {
            len += 4;
        } else if (u <= 0x3FFFFFF) {
            len += 5;
        } else {
            len += 6;
        }
    }
    return len;
}

// Given a pointer 'ptr' to a null-terminated UTF16LE-encoded string in the emscripten HEAP, returns
// a copy of that string as a Javascript String object.

var UTF16Decoder = typeof TextDecoder !== 'undefined' ? new TextDecoder('utf-16le') : undefined;

function UTF16ToString(ptr) {
    assert(ptr % 2 == 0, 'Pointer passed to UTF16ToString must be aligned to two bytes!');
    var endPtr = ptr;
    // TextDecoder needs to know the byte length in advance, it doesn't stop on null terminator by itself.
    // Also, use the length info to avoid running tiny strings through TextDecoder, since .subarray() allocates garbage.
    var idx = endPtr >> 1;
    while (HEAP16[idx]) ++idx;
    endPtr = idx << 1;

    if (endPtr - ptr > 32 && UTF16Decoder) {
        return UTF16Decoder.decode(HEAPU8.subarray(ptr, endPtr));
    } else {
        var i = 0;

        var str = '';
        while (1) {
            var codeUnit = HEAP16[(((ptr) + (i * 2)) >> 1)];
            if (codeUnit == 0) return str;
            ++i;
            // fromCharCode constructs a character from a UTF-16 code unit, so we can pass the UTF16 string right through.
            str += String.fromCharCode(codeUnit);
        }
    }
}

// Copies the given Javascript String object 'str' to the emscripten HEAP at address 'outPtr',
// null-terminated and encoded in UTF16 form. The copy will require at most str.length*4+2 bytes of space in the HEAP.
// Use the function lengthBytesUTF16() to compute the exact number of bytes (excluding null terminator) that this function will write.
// Parameters:
//   str: the Javascript string to copy.
//   outPtr: Byte address in Emscripten HEAP where to write the string to.
//   maxBytesToWrite: The maximum number of bytes this function can write to the array. This count should include the null
//                    terminator, i.e. if maxBytesToWrite=2, only the null terminator will be written and nothing else.
//                    maxBytesToWrite<2 does not write any bytes to the output, not even the null terminator.
// Returns the number of bytes written, EXCLUDING the null terminator.

function stringToUTF16(str, outPtr, maxBytesToWrite) {
    assert(outPtr % 2 == 0, 'Pointer passed to stringToUTF16 must be aligned to two bytes!');
    assert(typeof maxBytesToWrite == 'number', 'stringToUTF16(str, outPtr, maxBytesToWrite) is missing the third parameter that specifies the length of the output buffer!');
    // Backwards compatibility: if max bytes is not specified, assume unsafe unbounded write is allowed.
    if (maxBytesToWrite === undefined) {
        maxBytesToWrite = 0x7FFFFFFF;
    }
    if (maxBytesToWrite < 2) return 0;
    maxBytesToWrite -= 2; // Null terminator.
    var startPtr = outPtr;
    var numCharsToWrite = (maxBytesToWrite < str.length * 2) ? (maxBytesToWrite / 2) : str.length;
    for (var i = 0; i < numCharsToWrite; ++i) {
        // charCodeAt returns a UTF-16 encoded code unit, so it can be directly written to the HEAP.
        var codeUnit = str.charCodeAt(i); // possibly a lead surrogate
        HEAP16[((outPtr) >> 1)] = codeUnit;
        outPtr += 2;
    }
    // Null-terminate the pointer to the HEAP.
    HEAP16[((outPtr) >> 1)] = 0;
    return outPtr - startPtr;
}

// Returns the number of bytes the given Javascript string takes if encoded as a UTF16 byte array, EXCLUDING the null terminator byte.

function lengthBytesUTF16(str) {
    return str.length * 2;
}

function UTF32ToString(ptr) {
    assert(ptr % 4 == 0, 'Pointer passed to UTF32ToString must be aligned to four bytes!');
    var i = 0;

    var str = '';
    while (1) {
        var utf32 = HEAP32[(((ptr) + (i * 4)) >> 2)];
        if (utf32 == 0)
            return str;
        ++i;
        // Gotcha: fromCharCode constructs a character from a UTF-16 encoded code (pair), not from a Unicode code point! So encode the code point to UTF-16 for constructing.
        // See http://unicode.org/faq/utf_bom.html#utf16-3
        if (utf32 >= 0x10000) {
            var ch = utf32 - 0x10000;
            str += String.fromCharCode(0xD800 | (ch >> 10), 0xDC00 | (ch & 0x3FF));
        } else {
            str += String.fromCharCode(utf32);
        }
    }
}

// Copies the given Javascript String object 'str' to the emscripten HEAP at address 'outPtr',
// null-terminated and encoded in UTF32 form. The copy will require at most str.length*4+4 bytes of space in the HEAP.
// Use the function lengthBytesUTF32() to compute the exact number of bytes (excluding null terminator) that this function will write.
// Parameters:
//   str: the Javascript string to copy.
//   outPtr: Byte address in Emscripten HEAP where to write the string to.
//   maxBytesToWrite: The maximum number of bytes this function can write to the array. This count should include the null
//                    terminator, i.e. if maxBytesToWrite=4, only the null terminator will be written and nothing else.
//                    maxBytesToWrite<4 does not write any bytes to the output, not even the null terminator.
// Returns the number of bytes written, EXCLUDING the null terminator.

function stringToUTF32(str, outPtr, maxBytesToWrite) {
    assert(outPtr % 4 == 0, 'Pointer passed to stringToUTF32 must be aligned to four bytes!');
    assert(typeof maxBytesToWrite == 'number', 'stringToUTF32(str, outPtr, maxBytesToWrite) is missing the third parameter that specifies the length of the output buffer!');
    // Backwards compatibility: if max bytes is not specified, assume unsafe unbounded write is allowed.
    if (maxBytesToWrite === undefined) {
        maxBytesToWrite = 0x7FFFFFFF;
    }
    if (maxBytesToWrite < 4) return 0;
    var startPtr = outPtr;
    var endPtr = startPtr + maxBytesToWrite - 4;
    for (var i = 0; i < str.length; ++i) {
        // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code unit, not a Unicode code point of the character! We must decode the string to UTF-32 to the heap.
        // See http://unicode.org/faq/utf_bom.html#utf16-3
        var codeUnit = str.charCodeAt(i); // possibly a lead surrogate
        if (codeUnit >= 0xD800 && codeUnit <= 0xDFFF) {
            var trailSurrogate = str.charCodeAt(++i);
            codeUnit = 0x10000 + ((codeUnit & 0x3FF) << 10) | (trailSurrogate & 0x3FF);
        }
        HEAP32[((outPtr) >> 2)] = codeUnit;
        outPtr += 4;
        if (outPtr + 4 > endPtr) break;
    }
    // Null-terminate the pointer to the HEAP.
    HEAP32[((outPtr) >> 2)] = 0;
    return outPtr - startPtr;
}

// Returns the number of bytes the given Javascript string takes if encoded as a UTF16 byte array, EXCLUDING the null terminator byte.

function lengthBytesUTF32(str) {
    var len = 0;
    for (var i = 0; i < str.length; ++i) {
        // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code unit, not a Unicode code point of the character! We must decode the string to UTF-32 to the heap.
        // See http://unicode.org/faq/utf_bom.html#utf16-3
        var codeUnit = str.charCodeAt(i);
        if (codeUnit >= 0xD800 && codeUnit <= 0xDFFF) ++i; // possibly a lead surrogate, so skip over the tail surrogate.
        len += 4;
    }

    return len;
}

// Allocate heap space for a JS string, and write it there.
// It is the responsibility of the caller to free() that memory.
function allocateUTF8(str) {
    var size = lengthBytesUTF8(str) + 1;
    var ret = _malloc(size);
    if (ret) stringToUTF8Array(str, HEAP8, ret, size);
    return ret;
}

// Allocate stack space for a JS string, and write it there.
function allocateUTF8OnStack(str) {
    var size = lengthBytesUTF8(str) + 1;
    var ret = stackAlloc(size);
    stringToUTF8Array(str, HEAP8, ret, size);
    return ret;
}

function demangle(func) {
    warnOnce('warning: build with  -s DEMANGLE_SUPPORT=1  to link in libcxxabi demangling');
    return func;
}

function demangleAll(text) {
    var regex =
        /__Z[\w\d_]+/g;
    return text.replace(regex,
        function (x) {
            var y = demangle(x);
            return x === y ? x : (x + ' [' + y + ']');
        });
}

function jsStackTrace() {
    var err = new Error();
    if (!err.stack) {
        // IE10+ special cases: It does have callstack info, but it is only populated if an Error object is thrown,
        // so try that as a special-case.
        try {
            throw new Error(0);
        } catch (e) {
            err = e;
        }
        if (!err.stack) {
            return '(no stack trace available)';
        }
    }
    return err.stack.toString();
}

function stackTrace() {
    var js = jsStackTrace();
    if (Module['extraStackTrace']) js += '\n' + Module['extraStackTrace']();
    return demangleAll(js);
}

// Memory management

var PAGE_SIZE = 16384;
var WASM_PAGE_SIZE = 65536;
var ASMJS_PAGE_SIZE = 16777216;
var MIN_TOTAL_MEMORY = 16777216;

function alignUp(x, multiple) {
    if (x % multiple > 0) {
        x += multiple - (x % multiple);
    }
    return x;
}

var HEAP,
    /** @type {ArrayBuffer} */
    buffer,
    /** @type {Int8Array} */
    HEAP8,
    /** @type {Uint8Array} */
    HEAPU8,
    /** @type {Int16Array} */
    HEAP16,
    /** @type {Uint16Array} */
    HEAPU16,
    /** @type {Int32Array} */
    HEAP32,
    /** @type {Uint32Array} */
    HEAPU32,
    /** @type {Float32Array} */
    HEAPF32,
    /** @type {Float64Array} */
    HEAPF64;

function updateGlobalBuffer(buf) {
    Module['buffer'] = buffer = buf;
}

function updateGlobalBufferViews() {
    Module['HEAP8'] = HEAP8 = new Int8Array(buffer);
    Module['HEAP16'] = HEAP16 = new Int16Array(buffer);
    Module['HEAP32'] = HEAP32 = new Int32Array(buffer);
    Module['HEAPU8'] = HEAPU8 = new Uint8Array(buffer);
    Module['HEAPU16'] = HEAPU16 = new Uint16Array(buffer);
    Module['HEAPU32'] = HEAPU32 = new Uint32Array(buffer);
    Module['HEAPF32'] = HEAPF32 = new Float32Array(buffer);
    Module['HEAPF64'] = HEAPF64 = new Float64Array(buffer);
}

var STATIC_BASE, STATICTOP, staticSealed; // static area
var STACK_BASE, STACKTOP, STACK_MAX; // stack area
var DYNAMIC_BASE, DYNAMICTOP_PTR; // dynamic area handled by sbrk

STATIC_BASE = STATICTOP = STACK_BASE = STACKTOP = STACK_MAX = DYNAMIC_BASE = DYNAMICTOP_PTR = 0;
staticSealed = false;


// Initializes the stack cookie. Called at the startup of main and at the startup of each thread in pthreads mode.
function writeStackCookie() {
    assert((STACK_MAX & 3) == 0);
    HEAPU32[(STACK_MAX >> 2) - 1] = 0x02135467;
    HEAPU32[(STACK_MAX >> 2) - 2] = 0x89BACDFE;
}

function checkStackCookie() {
    if (HEAPU32[(STACK_MAX >> 2) - 1] != 0x02135467 || HEAPU32[(STACK_MAX >> 2) - 2] != 0x89BACDFE) {
        abort('Stack overflow! Stack cookie has been overwritten, expected hex dwords 0x89BACDFE and 0x02135467, but received 0x' + HEAPU32[(STACK_MAX >> 2) - 2].toString(16) + ' ' + HEAPU32[(STACK_MAX >> 2) - 1].toString(16));
    }
    // Also test the global address 0 for integrity. This check is not compatible with SAFE_SPLIT_MEMORY though, since that mode already tests all address 0 accesses on its own.
    if (HEAP32[0] !== 0x63736d65 /* 'emsc' */) throw 'Runtime error: The application has corrupted its heap memory area (address zero)!';
}

function abortStackOverflow(allocSize) {
    abort('Stack overflow! Attempted to allocate ' + allocSize + ' bytes on the stack, but stack has only ' + (STACK_MAX - stackSave() + allocSize) + ' bytes available!');
}


function abortOnCannotGrowMemory() {
    abort('Cannot enlarge memory arrays. Either (1) compile with  -s TOTAL_MEMORY=X  with X higher than the current value ' + TOTAL_MEMORY + ', (2) compile with  -s ALLOW_MEMORY_GROWTH=1  which allows increasing the size at runtime, or (3) if you want malloc to return NULL (0) instead of this abort, compile with  -s ABORTING_MALLOC=0 ');
}


function enlargeMemory() {
    abortOnCannotGrowMemory();
}


var TOTAL_STACK = Module['TOTAL_STACK'] || 5242880;
var TOTAL_MEMORY = Module['TOTAL_MEMORY'] || 16777216;
if (TOTAL_MEMORY < TOTAL_STACK) err('TOTAL_MEMORY should be larger than TOTAL_STACK, was ' + TOTAL_MEMORY + '! (TOTAL_STACK=' + TOTAL_STACK + ')');

// Initialize the runtime's memory
// check for full engine support (use string 'subarray' to avoid closure compiler confusion)
assert(typeof Int32Array !== 'undefined' && typeof Float64Array !== 'undefined' && Int32Array.prototype.subarray !== undefined && Int32Array.prototype.set !== undefined,
    'JS engine does not provide full typed array support');


// Use a provided buffer, if there is one, or else allocate a new one
if (Module['buffer']) {
    buffer = Module['buffer'];
    assert(buffer.byteLength === TOTAL_MEMORY, 'provided buffer should be ' + TOTAL_MEMORY + ' bytes, but it is ' + buffer.byteLength);
} else {
    // Use a WebAssembly memory where available
    if (typeof WebAssembly === 'object' && typeof WebAssembly.Memory === 'function') {
        assert(TOTAL_MEMORY % WASM_PAGE_SIZE === 0);
        Module['wasmMemory'] = new WebAssembly.Memory({
            'initial': TOTAL_MEMORY / WASM_PAGE_SIZE,
            'maximum': TOTAL_MEMORY / WASM_PAGE_SIZE
        });
        buffer = Module['wasmMemory'].buffer;
    } else {
        buffer = new ArrayBuffer(TOTAL_MEMORY);
    }
    assert(buffer.byteLength === TOTAL_MEMORY);
    Module['buffer'] = buffer;
}
updateGlobalBufferViews();


function getTotalMemory() {
    return TOTAL_MEMORY;
}

// Endianness check (note: assumes compiler arch was little-endian)
HEAP32[0] = 0x63736d65;
/* 'emsc' */
HEAP16[1] = 0x6373;
if (HEAPU8[2] !== 0x73 || HEAPU8[3] !== 0x63) throw 'Runtime error: expected the system to be little-endian!';

function callRuntimeCallbacks(callbacks) {
    while (callbacks.length > 0) {
        var callback = callbacks.shift();
        if (typeof callback == 'function') {
            callback();
            continue;
        }
        var func = callback.func;
        if (typeof func === 'number') {
            if (callback.arg === undefined) {
                Module['dynCall_v'](func);
            } else {
                Module['dynCall_vi'](func, callback.arg);
            }
        } else {
            func(callback.arg === undefined ? null : callback.arg);
        }
    }
}

var __ATPRERUN__ = []; // functions called before the runtime is initialized
var __ATINIT__ = []; // functions called during startup
var __ATMAIN__ = []; // functions called when main() is to be run
var __ATEXIT__ = []; // functions called during shutdown
var __ATPOSTRUN__ = []; // functions called after the main() is called

var runtimeInitialized = false;
var runtimeExited = false;


function preRun() {
    // compatibility - merge in anything from Module['preRun'] at this time
    if (Module['preRun']) {
        if (typeof Module['preRun'] == 'function') Module['preRun'] = [Module['preRun']];
        while (Module['preRun'].length) {
            addOnPreRun(Module['preRun'].shift());
        }
    }
    callRuntimeCallbacks(__ATPRERUN__);
}

function ensureInitRuntime() {
    checkStackCookie();
    if (runtimeInitialized) return;
    runtimeInitialized = true;
    callRuntimeCallbacks(__ATINIT__);
}

function preMain() {
    checkStackCookie();
    callRuntimeCallbacks(__ATMAIN__);
}

function exitRuntime() {
    checkStackCookie();
    callRuntimeCallbacks(__ATEXIT__);
    runtimeExited = true;
}

function postRun() {
    checkStackCookie();
    // compatibility - merge in anything from Module['postRun'] at this time
    if (Module['postRun']) {
        if (typeof Module['postRun'] == 'function') Module['postRun'] = [Module['postRun']];
        while (Module['postRun'].length) {
            addOnPostRun(Module['postRun'].shift());
        }
    }
    callRuntimeCallbacks(__ATPOSTRUN__);
}

function addOnPreRun(cb) {
    __ATPRERUN__.unshift(cb);
}

function addOnInit(cb) {
    __ATINIT__.unshift(cb);
}

function addOnPreMain(cb) {
    __ATMAIN__.unshift(cb);
}

function addOnExit(cb) {
    __ATEXIT__.unshift(cb);
}

function addOnPostRun(cb) {
    __ATPOSTRUN__.unshift(cb);
}

// Deprecated: This function should not be called because it is unsafe and does not provide
// a maximum length limit of how many bytes it is allowed to write. Prefer calling the
// function stringToUTF8Array() instead, which takes in a maximum length that can be used
// to be secure from out of bounds writes.
/** @deprecated */
function writeStringToMemory(string, buffer, dontAddNull) {
    warnOnce('writeStringToMemory is deprecated and should not be called! Use stringToUTF8() instead!');

    var /** @type {number} */ lastChar, /** @type {number} */ end;
    if (dontAddNull) {
        // stringToUTF8Array always appends null. If we don't want to do that, remember the
        // character that existed at the location where the null will be placed, and restore
        // that after the write (below).
        end = buffer + lengthBytesUTF8(string);
        lastChar = HEAP8[end];
    }
    stringToUTF8(string, buffer, Infinity);
    if (dontAddNull) HEAP8[end] = lastChar; // Restore the value under the null character.
}

function writeArrayToMemory(array, buffer) {
    assert(array.length >= 0, 'writeArrayToMemory array must have a length (should be an array or typed array)')
    HEAP8.set(array, buffer);
}

function writeAsciiToMemory(str, buffer, dontAddNull) {
    for (var i = 0; i < str.length; ++i) {
        assert(str.charCodeAt(i) === str.charCodeAt(i) & 0xff);
        HEAP8[((buffer++) >> 0)] = str.charCodeAt(i);
    }
    // Null-terminate the pointer to the HEAP.
    if (!dontAddNull) HEAP8[((buffer) >> 0)] = 0;
}

function unSign(value, bits, ignore) {
    if (value >= 0) {
        return value;
    }
    return bits <= 32 ? 2 * Math.abs(1 << (bits - 1)) + value // Need some trickery, since if bits == 32, we are right at the limit of the bits JS uses in bitshifts
        : Math.pow(2, bits) + value;
}

function reSign(value, bits, ignore) {
    if (value <= 0) {
        return value;
    }
    var half = bits <= 32 ? Math.abs(1 << (bits - 1)) // abs is needed if bits == 32
        : Math.pow(2, bits - 1);
    if (value >= half && (bits <= 32 || value > half)) { // for huge values, we can hit the precision limit and always get true here. so don't do that
        // but, in general there is no perfect solution here. With 64-bit ints, we get rounding and errors
        // TODO: In i64 mode 1, resign the two parts separately and safely
        value = -2 * half + value; // Cannot bitshift half, as it may be at the limit of the bits JS uses in bitshifts
    }
    return value;
}

assert(Math['imul'] && Math['fround'] && Math['clz32'] && Math['trunc'], 'this is a legacy browser, build with LEGACY_VM_SUPPORT');

var Math_abs = Math.abs;
var Math_cos = Math.cos;
var Math_sin = Math.sin;
var Math_tan = Math.tan;
var Math_acos = Math.acos;
var Math_asin = Math.asin;
var Math_atan = Math.atan;
var Math_atan2 = Math.atan2;
var Math_exp = Math.exp;
var Math_log = Math.log;
var Math_sqrt = Math.sqrt;
var Math_ceil = Math.ceil;
var Math_floor = Math.floor;
var Math_pow = Math.pow;
var Math_imul = Math.imul;
var Math_fround = Math.fround;
var Math_round = Math.round;
var Math_min = Math.min;
var Math_max = Math.max;
var Math_clz32 = Math.clz32;
var Math_trunc = Math.trunc;

// A counter of dependencies for calling run(). If we need to
// do asynchronous work before running, increment this and
// decrement it. Incrementing must happen in a place like
// PRE_RUN_ADDITIONS (used by emcc to add file preloading).
// Note that you can add dependencies in preRun, even though
// it happens right before run - run will be postponed until
// the dependencies are met.
var runDependencies = 0;
var runDependencyWatcher = null;
var dependenciesFulfilled = null; // overridden to take different actions when all run dependencies are fulfilled
var runDependencyTracking = {};

function getUniqueRunDependency(id) {
    var orig = id;
    while (1) {
        if (!runDependencyTracking[id]) return id;
        id = orig + Math.random();
    }
    return id;
}

function addRunDependency(id) {
    runDependencies++;
    if (Module['monitorRunDependencies']) {
        Module['monitorRunDependencies'](runDependencies);
    }
    if (id) {
        assert(!runDependencyTracking[id]);
        runDependencyTracking[id] = 1;
        if (runDependencyWatcher === null && typeof setInterval !== 'undefined') {
            // Check for missing dependencies every few seconds
            runDependencyWatcher = setInterval(function () {
                if (ABORT) {
                    clearInterval(runDependencyWatcher);
                    runDependencyWatcher = null;
                    return;
                }
                var shown = false;
                for (var dep in runDependencyTracking) {
                    if (!shown) {
                        shown = true;
                        err('still waiting on run dependencies:');
                    }
                    err('dependency: ' + dep);
                }
                if (shown) {
                    err('(end of list)');
                }
            }, 10000);
        }
    } else {
        err('warning: run dependency added without ID');
    }
}

function removeRunDependency(id) {
    runDependencies--;
    if (Module['monitorRunDependencies']) {
        Module['monitorRunDependencies'](runDependencies);
    }
    if (id) {
        assert(runDependencyTracking[id]);
        delete runDependencyTracking[id];
    } else {
        err('warning: run dependency removed without ID');
    }
    if (runDependencies == 0) {
        if (runDependencyWatcher !== null) {
            clearInterval(runDependencyWatcher);
            runDependencyWatcher = null;
        }
        if (dependenciesFulfilled) {
            var callback = dependenciesFulfilled;
            dependenciesFulfilled = null;
            callback(); // can add another dependenciesFulfilled
        }
    }
}

Module["preloadedImages"] = {}; // maps url to image data
Module["preloadedAudios"] = {}; // maps url to audio data


var memoryInitializer = null;


var /* show errors on likely calls to FS when it was not included */ FS = {
    error: function () {
        abort('Filesystem support (FS) was not included. The problem is that you are using files from JS, but files were not used from C/C++, so filesystem support was not auto-included. You can force-include filesystem support with  -s FORCE_FILESYSTEM=1');
    },
    init: function () {
        FS.error()
    },
    createDataFile: function () {
        FS.error()
    },
    createPreloadedFile: function () {
        FS.error()
    },
    createLazyFile: function () {
        FS.error()
    },
    open: function () {
        FS.error()
    },
    mkdev: function () {
        FS.error()
    },
    registerDevice: function () {
        FS.error()
    },
    analyzePath: function () {
        FS.error()
    },
    loadFilesFromDB: function () {
        FS.error()
    },

    ErrnoError: function ErrnoError() {
        FS.error()
    },
};
Module['FS_createDataFile'] = FS.createDataFile;
Module['FS_createPreloadedFile'] = FS.createPreloadedFile;


// Prefix of data URIs emitted by SINGLE_FILE and related options.
var dataURIPrefix = 'data:application/octet-stream;base64,';

// Indicates whether filename is a base64 data URI.
function isDataURI(filename) {
    return String.prototype.startsWith ?
        filename.startsWith(dataURIPrefix) :
        filename.indexOf(dataURIPrefix) === 0;
}


function integrateWasmJS() {
    // wasm.js has several methods for creating the compiled code module here:
    //  * 'native-wasm' : use native WebAssembly support in the browser
    //  * 'interpret-s-expr': load s-expression code from a .wast and interpret
    //  * 'interpret-binary': load binary wasm and interpret
    //  * 'interpret-asm2wasm': load asm.js code, translate to wasm, and interpret
    //  * 'asmjs': no wasm, just load the asm.js code and use that (good for testing)
    // The method is set at compile time (BINARYEN_METHOD)
    // The method can be a comma-separated list, in which case, we will try the
    // options one by one. Some of them can fail gracefully, and then we can try
    // the next.

    // inputs

    var method = 'native-wasm';

    var wasmTextFile = 'hello.wast';
    var wasmBinaryFile = 'hello.wasm';
    var asmjsCodeFile = 'hello.temp.asm.js';

    if (!isDataURI(wasmTextFile)) {
        wasmTextFile = locateFile(wasmTextFile);
    }
    if (!isDataURI(wasmBinaryFile)) {
        wasmBinaryFile = locateFile(wasmBinaryFile);
    }
    if (!isDataURI(asmjsCodeFile)) {
        asmjsCodeFile = locateFile(asmjsCodeFile);
    }

    // utilities

    var wasmPageSize = 64 * 1024;

    var info = {
        'global': null,
        'env': null,
        'asm2wasm': asm2wasmImports,
        'parent': Module // Module inside wasm-js.cpp refers to wasm-js.cpp; this allows access to the outside program.
    };

    var exports = null;


    function mergeMemory(newBuffer) {
        // The wasm instance creates its memory. But static init code might have written to
        // buffer already, including the mem init file, and we must copy it over in a proper merge.
        // TODO: avoid this copy, by avoiding such static init writes
        // TODO: in shorter term, just copy up to the last static init write
        var oldBuffer = Module['buffer'];
        if (newBuffer.byteLength < oldBuffer.byteLength) {
            err('the new buffer in mergeMemory is smaller than the previous one. in native wasm, we should grow memory here');
        }
        var oldView = new Int8Array(oldBuffer);
        var newView = new Int8Array(newBuffer);


        newView.set(oldView);
        updateGlobalBuffer(newBuffer);
        updateGlobalBufferViews();
    }

    function fixImports(imports) {
        return imports;
    }

    function getBinary() {
        try {
            if (Module['wasmBinary']) {
                return new Uint8Array(Module['wasmBinary']);
            }
            if (Module['readBinary']) {
                return Module['readBinary'](wasmBinaryFile);
            } else {
                throw "both async and sync fetching of the wasm failed";
            }
        }
        catch (err) {
            abort(err);
        }
    }

    function getBinaryPromise() {
        // if we don't have the binary yet, and have the Fetch api, use that
        // in some environments, like Electron's render process, Fetch api may be present, but have a different context than expected, let's only use it on the Web
        if (!Module['wasmBinary'] && (ENVIRONMENT_IS_WEB || ENVIRONMENT_IS_WORKER) && typeof fetch === 'function') {
            return fetch(wasmBinaryFile, {credentials: 'same-origin'}).then(function (response) {
                if (!response['ok']) {
                    throw "failed to load wasm binary file at '" + wasmBinaryFile + "'";
                }
                return response['arrayBuffer']();
            }).catch(function () {
                return getBinary();
            });
        }
        // Otherwise, getBinary should be able to get it synchronously
        return new Promise(function (resolve, reject) {
            resolve(getBinary());
        });
    }

    // do-method functions


    function doNativeWasm(global, env, providedBuffer) {
        if (typeof WebAssembly !== 'object') {
            // when the method is just native-wasm, our error message can be very specific
            abort('No WebAssembly support found. Build with -s WASM=0 to target JavaScript instead.');
            err('no native wasm support detected');
            return false;
        }
        // prepare memory import
        if (!(Module['wasmMemory'] instanceof WebAssembly.Memory)) {
            err('no native wasm Memory in use');
            return false;
        }
        env['memory'] = Module['wasmMemory'];
        // Load the wasm module and create an instance of using native support in the JS engine.
        info['global'] = {
            'NaN': NaN,
            'Infinity': Infinity
        };
        info['global.Math'] = Math;
        info['env'] = env;
        // handle a generated wasm instance, receiving its exports and
        // performing other necessary setup
        function receiveInstance(instance, module) {
            exports = instance.exports;
            if (exports.memory) mergeMemory(exports.memory);
            Module['asm'] = exports;
            Module["usingWasm"] = true;
            removeRunDependency('wasm-instantiate');
        }

        addRunDependency('wasm-instantiate');

        // User shell pages can write their own Module.instantiateWasm = function(imports, successCallback) callback
        // to manually instantiate the Wasm module themselves. This allows pages to run the instantiation parallel
        // to any other async startup actions they are performing.
        if (Module['instantiateWasm']) {
            try {
                return Module['instantiateWasm'](info, receiveInstance);
            } catch (e) {
                err('Module.instantiateWasm callback failed with error: ' + e);
                return false;
            }
        }

        // Async compilation can be confusing when an error on the page overwrites Module
        // (for example, if the order of elements is wrong, and the one defining Module is
        // later), so we save Module and check it later.
        var trueModule = Module;

        function receiveInstantiatedSource(output) {
            // 'output' is a WebAssemblyInstantiatedSource object which has both the module and instance.
            // receiveInstance() will swap in the exports (to Module.asm) so they can be called
            assert(Module === trueModule, 'the Module object should not be replaced during async compilation - perhaps the order of HTML elements is wrong?');
            trueModule = null;
            receiveInstance(output['instance'], output['module']);
        }

        function instantiateArrayBuffer(receiver) {
            getBinaryPromise().then(function (binary) {
                return WebAssembly.instantiate(binary, info);
            }).then(receiver).catch(function (reason) {
                err('failed to asynchronously prepare wasm: ' + reason);
                abort(reason);
            });
        }

        // Prefer streaming instantiation if available.
        if (!Module['wasmBinary'] &&
            typeof WebAssembly.instantiateStreaming === 'function' &&
            !isDataURI(wasmBinaryFile) &&
            typeof fetch === 'function') {
            WebAssembly.instantiateStreaming(fetch(wasmBinaryFile, {credentials: 'same-origin'}), info)
                .then(receiveInstantiatedSource)
                .catch(function (reason) {
                    // We expect the most common failure cause to be a bad MIME type for the binary,
                    // in which case falling back to ArrayBuffer instantiation should work.
                    err('wasm streaming compile failed: ' + reason);
                    err('falling back to ArrayBuffer instantiation');
                    instantiateArrayBuffer(receiveInstantiatedSource);
                });
        } else {
            instantiateArrayBuffer(receiveInstantiatedSource);
        }
        return {}; // no exports yet; we'll fill them in later
    }


    // We may have a preloaded value in Module.asm, save it
    Module['asmPreload'] = Module['asm'];

    // Memory growth integration code

    var asmjsReallocBuffer = Module['reallocBuffer'];

    var wasmReallocBuffer = function (size) {
        var PAGE_MULTIPLE = Module["usingWasm"] ? WASM_PAGE_SIZE : ASMJS_PAGE_SIZE; // In wasm, heap size must be a multiple of 64KB. In asm.js, they need to be multiples of 16MB.
        size = alignUp(size, PAGE_MULTIPLE); // round up to wasm page size
        var old = Module['buffer'];
        var oldSize = old.byteLength;
        if (Module["usingWasm"]) {
            // native wasm support
            try {
                var result = Module['wasmMemory'].grow((size - oldSize) / wasmPageSize); // .grow() takes a delta compared to the previous size
                if (result !== (-1 | 0)) {
                    // success in native wasm memory growth, get the buffer from the memory
                    return Module['buffer'] = Module['wasmMemory'].buffer;
                } else {
                    return null;
                }
            } catch (e) {
                console.error('Module.reallocBuffer: Attempted to grow from ' + oldSize + ' bytes to ' + size + ' bytes, but got error: ' + e);
                return null;
            }
        }
    };

    Module['reallocBuffer'] = function (size) {
        if (finalMethod === 'asmjs') {
            return asmjsReallocBuffer(size);
        } else {
            return wasmReallocBuffer(size);
        }
    };

    // we may try more than one; this is the final one, that worked and we are using
    var finalMethod = '';

    // Provide an "asm.js function" for the application, called to "link" the asm.js module. We instantiate
    // the wasm module at that time, and it receives imports and provides exports and so forth, the app
    // doesn't need to care that it is wasm or olyfilled wasm or asm.js.

    Module['asm'] = function (global, env, providedBuffer) {
        env = fixImports(env);

        // import table
        if (!env['table']) {
            var TABLE_SIZE = Module['wasmTableSize'];
            if (TABLE_SIZE === undefined) TABLE_SIZE = 1024; // works in binaryen interpreter at least
            var MAX_TABLE_SIZE = Module['wasmMaxTableSize'];
            if (typeof WebAssembly === 'object' && typeof WebAssembly.Table === 'function') {
                if (MAX_TABLE_SIZE !== undefined) {
                    env['table'] = new WebAssembly.Table({
                        'initial': TABLE_SIZE,
                        'maximum': MAX_TABLE_SIZE,
                        'element': 'anyfunc'
                    });
                } else {
                    env['table'] = new WebAssembly.Table({'initial': TABLE_SIZE, element: 'anyfunc'});
                }
            } else {
                env['table'] = new Array(TABLE_SIZE); // works in binaryen interpreter at least
            }
            Module['wasmTable'] = env['table'];
        }

        if (!env['memoryBase']) {
            env['memoryBase'] = Module['STATIC_BASE']; // tell the memory segments where to place themselves
        }
        if (!env['tableBase']) {
            env['tableBase'] = 0; // table starts at 0 by default, in dynamic linking this will change
        }

        // try the methods. each should return the exports if it succeeded

        var exports;
        exports = doNativeWasm(global, env, providedBuffer);

        assert(exports, 'no binaryen method succeeded. consider enabling more options, like interpreting, if you want that: https://github.com/kripken/emscripten/wiki/WebAssembly#binaryen-methods');


        return exports;
    };

    var methodHandler = Module['asm']; // note our method handler, as we may modify Module['asm'] later
}

integrateWasmJS();

// === Body ===

var ASM_CONSTS = [];


STATIC_BASE = GLOBAL_BASE;

STATICTOP = STATIC_BASE + 3168;
/* global initializers */
__ATINIT__.push();


var STATIC_BUMP = 3168;
Module["STATIC_BASE"] = STATIC_BASE;
Module["STATIC_BUMP"] = STATIC_BUMP;

/* no memory initializer */
var tempDoublePtr = STATICTOP;
STATICTOP += 16;

assert(tempDoublePtr % 8 == 0);

function copyTempFloat(ptr) { // functions, because inlining this code increases code size too much

    HEAP8[tempDoublePtr] = HEAP8[ptr];

    HEAP8[tempDoublePtr + 1] = HEAP8[ptr + 1];

    HEAP8[tempDoublePtr + 2] = HEAP8[ptr + 2];

    HEAP8[tempDoublePtr + 3] = HEAP8[ptr + 3];

}

function copyTempDouble(ptr) {

    HEAP8[tempDoublePtr] = HEAP8[ptr];

    HEAP8[tempDoublePtr + 1] = HEAP8[ptr + 1];

    HEAP8[tempDoublePtr + 2] = HEAP8[ptr + 2];

    HEAP8[tempDoublePtr + 3] = HEAP8[ptr + 3];

    HEAP8[tempDoublePtr + 4] = HEAP8[ptr + 4];

    HEAP8[tempDoublePtr + 5] = HEAP8[ptr + 5];

    HEAP8[tempDoublePtr + 6] = HEAP8[ptr + 6];

    HEAP8[tempDoublePtr + 7] = HEAP8[ptr + 7];

}

// {{PRE_LIBRARY}}


function ___cxa_allocate_exception(size) {
    return _malloc(size);
}


function __ZSt18uncaught_exceptionv() { // std::uncaught_exception()
    return !!__ZSt18uncaught_exceptionv.uncaught_exception;
}

var EXCEPTIONS = {
    last: 0, caught: [], infos: {}, deAdjust: function (adjusted) {
        if (!adjusted || EXCEPTIONS.infos[adjusted]) return adjusted;
        for (var key in EXCEPTIONS.infos) {
            var ptr = +key; // the iteration key is a string, and if we throw this, it must be an integer as that is what we look for
            var info = EXCEPTIONS.infos[ptr];
            if (info.adjusted === adjusted) {
                return ptr;
            }
        }
        return adjusted;
    }, addRef: function (ptr) {
        if (!ptr) return;
        var info = EXCEPTIONS.infos[ptr];
        info.refcount++;
    }, decRef: function (ptr) {
        if (!ptr) return;
        var info = EXCEPTIONS.infos[ptr];
        assert(info.refcount > 0);
        info.refcount--;
        // A rethrown exception can reach refcount 0; it must not be discarded
        // Its next handler will clear the rethrown flag and addRef it, prior to
        // final decRef and destruction here
        if (info.refcount === 0 && !info.rethrown) {
            if (info.destructor) {
                Module['dynCall_vi'](info.destructor, ptr);
            }
            delete EXCEPTIONS.infos[ptr];
            ___cxa_free_exception(ptr);
        }
    }, clearRef: function (ptr) {
        if (!ptr) return;
        var info = EXCEPTIONS.infos[ptr];
        info.refcount = 0;
    }
};

function ___cxa_begin_catch(ptr) {
    var info = EXCEPTIONS.infos[ptr];
    if (info && !info.caught) {
        info.caught = true;
        __ZSt18uncaught_exceptionv.uncaught_exception--;
    }
    if (info) info.rethrown = false;
    EXCEPTIONS.caught.push(ptr);
    EXCEPTIONS.addRef(EXCEPTIONS.deAdjust(ptr));
    return ptr;
}


function ___cxa_free_exception(ptr) {
    try {
        return _free(ptr);
    } catch (e) { // XXX FIXME
        err('exception during cxa_free_exception: ' + e);
    }
}

function ___cxa_end_catch() {
    // Clear state flag.
    Module['setThrew'](0);
    // Call destructor if one is registered then clear it.
    var ptr = EXCEPTIONS.caught.pop();
    if (ptr) {
        EXCEPTIONS.decRef(EXCEPTIONS.deAdjust(ptr));
        EXCEPTIONS.last = 0; // XXX in decRef?
    }
}

function ___cxa_find_matching_catch_3() {
    console.log("3 ARGS: ", arguments)
    return ___cxa_find_matching_catch.apply(null, arguments);
}


function ___resumeException(ptr) {
    if (!EXCEPTIONS.last) {
        EXCEPTIONS.last = ptr;
    }
    throw ptr;
}

function ___cxa_find_matching_catch() {
    var thrown = EXCEPTIONS.last;
    if (!thrown) {
        // just pass through the null ptr
        return ((setTempRet0(0), 0) | 0);
    }
    var info = EXCEPTIONS.infos[thrown];
    var throwntype = info.type;
    if (!throwntype) {
        // just pass through the thrown ptr
        return ((setTempRet0(0), thrown) | 0);
    }
    var typeArray = Array.prototype.slice.call(arguments);

    console.log(arguments, typeArray)

    var pointer = Module['___cxa_is_pointer_type'](throwntype);
    // can_catch receives a **, add indirection
    if (!___cxa_find_matching_catch.buffer) ___cxa_find_matching_catch.buffer = _malloc(4);
    HEAP32[((___cxa_find_matching_catch.buffer) >> 2)] = thrown;
    thrown = ___cxa_find_matching_catch.buffer;
    // The different catch blocks are denoted by different types.
    // Due to inheritance, those types may not precisely match the
    // type of the thrown object. Find one which matches, and
    // return the type of the catch block which should be called.
    for (var i = 0; i < typeArray.length; i++) {
        if (typeArray[i] && Module['___cxa_can_catch'](typeArray[i], throwntype, thrown)) {
            thrown = HEAP32[((thrown) >> 2)]; // undo indirection
            info.adjusted = thrown;

            return ((setTempRet0(typeArray[i]), thrown) | 0);
        }
    }
    // Shouldn't happen unless we have bogus data in typeArray
    // or encounter a type for which emscripten doesn't have suitable
    // typeinfo defined. Best-efforts match just in case.
    thrown = HEAP32[((thrown) >> 2)]; // undo indirection
    return ((setTempRet0(throwntype), thrown) | 0);
}

function ___cxa_throw(ptr, type, destructor) {
    EXCEPTIONS.infos[ptr] = {
        ptr: ptr,
        adjusted: ptr,
        type: type,
        destructor: destructor,
        refcount: 0,
        caught: false,
        rethrown: false
    };
    EXCEPTIONS.last = ptr;
    if (!("uncaught_exception" in __ZSt18uncaught_exceptionv)) {
        __ZSt18uncaught_exceptionv.uncaught_exception = 1;
    } else {
        __ZSt18uncaught_exceptionv.uncaught_exception++;
    }
    throw ptr;
}

function ___gxx_personality_v0() {
}

function ___lock() {
}


var SYSCALLS = {
    varargs: 0, get: function (varargs) {
        SYSCALLS.varargs += 4;
        var ret = HEAP32[(((SYSCALLS.varargs) - (4)) >> 2)];
        return ret;
    }, getStr: function () {
        var ret = Pointer_stringify(SYSCALLS.get());
        return ret;
    }, get64: function () {
        var low = SYSCALLS.get(), high = SYSCALLS.get();
        if (low >= 0) assert(high === 0);
        else assert(high === -1);
        return low;
    }, getZero: function () {
        assert(SYSCALLS.get() === 0);
    }
};

function ___syscall140(which, varargs) {
    SYSCALLS.varargs = varargs;
    try {
        // llseek
        var stream = SYSCALLS.getStreamFromFD(), offset_high = SYSCALLS.get(), offset_low = SYSCALLS.get(),
            result = SYSCALLS.get(), whence = SYSCALLS.get();
        // NOTE: offset_high is unused - Emscripten's off_t is 32-bit
        var offset = offset_low;
        FS.llseek(stream, offset, whence);
        HEAP32[((result) >> 2)] = stream.position;
        if (stream.getdents && offset === 0 && whence === 0) stream.getdents = null; // reset readdir state
        return 0;
    } catch (e) {
        if (typeof FS === 'undefined' || !(e instanceof FS.ErrnoError)) abort(e);
        return -e.errno;
    }
}


function flush_NO_FILESYSTEM() {
    // flush anything remaining in the buffers during shutdown
    var fflush = Module["_fflush"];
    if (fflush) fflush(0);
    var printChar = ___syscall146.printChar;
    if (!printChar) return;
    var buffers = ___syscall146.buffers;
    if (buffers[1].length) printChar(1, 10);
    if (buffers[2].length) printChar(2, 10);
}

function ___syscall146(which, varargs) {
    SYSCALLS.varargs = varargs;
    try {
        // writev
        // hack to support printf in NO_FILESYSTEM
        var stream = SYSCALLS.get(), iov = SYSCALLS.get(), iovcnt = SYSCALLS.get();
        var ret = 0;
        if (!___syscall146.buffers) {
            ___syscall146.buffers = [null, [], []]; // 1 => stdout, 2 => stderr
            ___syscall146.printChar = function (stream, curr) {
                var buffer = ___syscall146.buffers[stream];
                assert(buffer);
                if (curr === 0 || curr === 10) {
                    (stream === 1 ? out : err)(UTF8ArrayToString(buffer, 0));
                    buffer.length = 0;
                } else {
                    buffer.push(curr);
                }
            };
        }
        for (var i = 0; i < iovcnt; i++) {
            var ptr = HEAP32[(((iov) + (i * 8)) >> 2)];
            var len = HEAP32[(((iov) + (i * 8 + 4)) >> 2)];
            for (var j = 0; j < len; j++) {
                ___syscall146.printChar(stream, HEAPU8[ptr + j]);
            }
            ret += len;
        }
        return ret;
    } catch (e) {
        if (typeof FS === 'undefined' || !(e instanceof FS.ErrnoError)) abort(e);
        return -e.errno;
    }
}

function ___syscall54(which, varargs) {
    SYSCALLS.varargs = varargs;
    try {
        // ioctl
        return 0;
    } catch (e) {
        if (typeof FS === 'undefined' || !(e instanceof FS.ErrnoError)) abort(e);
        return -e.errno;
    }
}

function ___syscall6(which, varargs) {
    SYSCALLS.varargs = varargs;
    try {
        // close
        var stream = SYSCALLS.getStreamFromFD();
        FS.close(stream);
        return 0;
    } catch (e) {
        if (typeof FS === 'undefined' || !(e instanceof FS.ErrnoError)) abort(e);
        return -e.errno;
    }
}

function ___unlock() {
}

function _llvm_eh_typeid_for(type) {
    return type;
}


function _emscripten_memcpy_big(dest, src, num) {
    HEAPU8.set(HEAPU8.subarray(src, src + num), dest);
    return dest;
}


function ___setErrNo(value) {
    if (Module['___errno_location']) HEAP32[((Module['___errno_location']()) >> 2)] = value;
    else err('failed to set errno from JS');
    return value;
}

DYNAMICTOP_PTR = staticAlloc(4);

STACK_BASE = STACKTOP = alignMemory(STATICTOP);

STACK_MAX = STACK_BASE + TOTAL_STACK;

DYNAMIC_BASE = alignMemory(STACK_MAX);

HEAP32[DYNAMICTOP_PTR >> 2] = DYNAMIC_BASE;

staticSealed = true; // seal the static portion of memory

assert(DYNAMIC_BASE < TOTAL_MEMORY, "TOTAL_MEMORY not big enough for stack");

var ASSERTIONS = true;

/** @type {function(string, boolean=, number=)} */
function intArrayFromString(stringy, dontAddNull, length) {
    var len = length > 0 ? length : lengthBytesUTF8(stringy) + 1;
    var u8array = new Array(len);
    var numBytesWritten = stringToUTF8Array(stringy, u8array, 0, u8array.length);
    if (dontAddNull) u8array.length = numBytesWritten;
    return u8array;
}

function intArrayToString(array) {
    var ret = [];
    for (var i = 0; i < array.length; i++) {
        var chr = array[i];
        if (chr > 0xFF) {
            if (ASSERTIONS) {
                assert(false, 'Character code ' + chr + ' (' + String.fromCharCode(chr) + ')  at offset ' + i + ' not in 0x00-0xFF.');
            }
            chr &= 0xFF;
        }
        ret.push(String.fromCharCode(chr));
    }
    return ret.join('');
}


function nullFunc_ii(x) {
    err("Invalid function pointer called with signature 'ii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_iiii(x) {
    err("Invalid function pointer called with signature 'iiii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_vi(x) {
    err("Invalid function pointer called with signature 'vi'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_viii(x) {
    err("Invalid function pointer called with signature 'viii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_viiii(x) {
    err("Invalid function pointer called with signature 'viiii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_viiiii(x) {
    err("Invalid function pointer called with signature 'viiiii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

function nullFunc_viiiiii(x) {
    err("Invalid function pointer called with signature 'viiiiii'. Perhaps this is an invalid value (e.g. caused by calling a virtual method on a NULL pointer)? Or calling a function with an incorrect type, which will fail? (it is worth building your source files with -Werror (warnings are errors), as warnings can indicate undefined behavior which can cause this)");
    err("Build with ASSERTIONS=2 for more info.");
    abort(x)
}

Module['wasmTableSize'] = 146;

Module['wasmMaxTableSize'] = 146;

function invoke_ii(index, a1) {
    var sp = stackSave();
    try {
        return Module["dynCall_ii"](index, a1);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_iiii(index, a1, a2, a3) {
    var sp = stackSave();
    try {
        return Module["dynCall_iiii"](index, a1, a2, a3);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_vi(index, a1) {
    var sp = stackSave();
    try {
        Module["dynCall_vi"](index, a1);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_viii(index, a1, a2, a3) {
    var sp = stackSave();
    try {
        Module["dynCall_viii"](index, a1, a2, a3);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_viiii(index, a1, a2, a3, a4) {
    var sp = stackSave();
    try {
        Module["dynCall_viiii"](index, a1, a2, a3, a4);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_viiiii(index, a1, a2, a3, a4, a5) {
    var sp = stackSave();
    try {
        Module["dynCall_viiiii"](index, a1, a2, a3, a4, a5);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

function invoke_viiiiii(index, a1, a2, a3, a4, a5, a6) {
    var sp = stackSave();
    try {
        Module["dynCall_viiiiii"](index, a1, a2, a3, a4, a5, a6);
    } catch (e) {
        stackRestore(sp);
        if (typeof e !== 'number' && e !== 'longjmp') throw e;
        Module["setThrew"](1, 0);
    }
}

Module.asmGlobalArg = {};

Module.asmLibraryArg = {
    "abort": abort,
    "assert": assert,
    "enlargeMemory": enlargeMemory,
    "getTotalMemory": getTotalMemory,
    "abortOnCannotGrowMemory": abortOnCannotGrowMemory,
    "abortStackOverflow": abortStackOverflow,
    "nullFunc_ii": nullFunc_ii,
    "nullFunc_iiii": nullFunc_iiii,
    "nullFunc_vi": nullFunc_vi,
    "nullFunc_viii": nullFunc_viii,
    "nullFunc_viiii": nullFunc_viiii,
    "nullFunc_viiiii": nullFunc_viiiii,
    "nullFunc_viiiiii": nullFunc_viiiiii,
    "invoke_ii": invoke_ii,
    "invoke_iiii": invoke_iiii,
    "invoke_vi": invoke_vi,
    "invoke_viii": invoke_viii,
    "invoke_viiii": invoke_viiii,
    "invoke_viiiii": invoke_viiiii,
    "invoke_viiiiii": invoke_viiiiii,
    "__ZSt18uncaught_exceptionv": __ZSt18uncaught_exceptionv,
    "___cxa_allocate_exception": ___cxa_allocate_exception,
    "___cxa_begin_catch": ___cxa_begin_catch,
    "___cxa_end_catch": ___cxa_end_catch,
    "___cxa_find_matching_catch": ___cxa_find_matching_catch,
    "___cxa_find_matching_catch_3": ___cxa_find_matching_catch_3,
    "___cxa_free_exception": ___cxa_free_exception,
    "___cxa_throw": ___cxa_throw,
    "___gxx_personality_v0": ___gxx_personality_v0,
    "___lock": ___lock,
    "___resumeException": ___resumeException,
    "___setErrNo": ___setErrNo,
    "___syscall140": ___syscall140,
    "___syscall146": ___syscall146,
    "___syscall54": ___syscall54,
    "___syscall6": ___syscall6,
    "___unlock": ___unlock,
    "_emscripten_memcpy_big": _emscripten_memcpy_big,
    "_llvm_eh_typeid_for": _llvm_eh_typeid_for,
    "flush_NO_FILESYSTEM": flush_NO_FILESYSTEM,
    "DYNAMICTOP_PTR": DYNAMICTOP_PTR,
    "tempDoublePtr": tempDoublePtr,
    "ABORT": ABORT,
    "STACKTOP": STACKTOP,
    "STACK_MAX": STACK_MAX
};
// EMSCRIPTEN_START_ASM
var asm = Module["asm"]// EMSCRIPTEN_END_ASM
(Module.asmGlobalArg, Module.asmLibraryArg, buffer);

var real____cxa_can_catch = asm["___cxa_can_catch"];
asm["___cxa_can_catch"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real____cxa_can_catch.apply(null, arguments);
};

var real____cxa_is_pointer_type = asm["___cxa_is_pointer_type"];
asm["___cxa_is_pointer_type"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real____cxa_is_pointer_type.apply(null, arguments);
};

var real____errno_location = asm["___errno_location"];
asm["___errno_location"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real____errno_location.apply(null, arguments);
};

var real__fflush = asm["_fflush"];
asm["_fflush"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real__fflush.apply(null, arguments);
};

var real__free = asm["_free"];
asm["_free"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real__free.apply(null, arguments);
};

var real__main = asm["_main"];
asm["_main"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real__main.apply(null, arguments);
};

var real__malloc = asm["_malloc"];
asm["_malloc"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real__malloc.apply(null, arguments);
};

var real__sbrk = asm["_sbrk"];
asm["_sbrk"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real__sbrk.apply(null, arguments);
};

var real_establishStackSpace = asm["establishStackSpace"];
asm["establishStackSpace"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_establishStackSpace.apply(null, arguments);
};

var real_getTempRet0 = asm["getTempRet0"];
asm["getTempRet0"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_getTempRet0.apply(null, arguments);
};

var real_setTempRet0 = asm["setTempRet0"];
asm["setTempRet0"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_setTempRet0.apply(null, arguments);
};

var real_setThrew = asm["setThrew"];
asm["setThrew"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_setThrew.apply(null, arguments);
};

var real_stackAlloc = asm["stackAlloc"];
asm["stackAlloc"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_stackAlloc.apply(null, arguments);
};

var real_stackRestore = asm["stackRestore"];
asm["stackRestore"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_stackRestore.apply(null, arguments);
};

var real_stackSave = asm["stackSave"];
asm["stackSave"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return real_stackSave.apply(null, arguments);
};
Module["asm"] = asm;
var ___cxa_can_catch = Module["___cxa_can_catch"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["___cxa_can_catch"].apply(null, arguments)
};
var ___cxa_is_pointer_type = Module["___cxa_is_pointer_type"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["___cxa_is_pointer_type"].apply(null, arguments)
};
var ___errno_location = Module["___errno_location"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["___errno_location"].apply(null, arguments)
};
var _fflush = Module["_fflush"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_fflush"].apply(null, arguments)
};
var _free = Module["_free"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_free"].apply(null, arguments)
};
var _main = Module["_main"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_main"].apply(null, arguments)
};
var _malloc = Module["_malloc"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_malloc"].apply(null, arguments)
};
var _memcpy = Module["_memcpy"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_memcpy"].apply(null, arguments)
};
var _memset = Module["_memset"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_memset"].apply(null, arguments)
};
var _sbrk = Module["_sbrk"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["_sbrk"].apply(null, arguments)
};
var establishStackSpace = Module["establishStackSpace"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["establishStackSpace"].apply(null, arguments)
};
var getTempRet0 = Module["getTempRet0"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["getTempRet0"].apply(null, arguments)
};
var runPostSets = Module["runPostSets"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["runPostSets"].apply(null, arguments)
};
var setTempRet0 = Module["setTempRet0"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["setTempRet0"].apply(null, arguments)
};
var setThrew = Module["setThrew"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["setThrew"].apply(null, arguments)
};
var stackAlloc = Module["stackAlloc"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["stackAlloc"].apply(null, arguments)
};
var stackRestore = Module["stackRestore"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["stackRestore"].apply(null, arguments)
};
var stackSave = Module["stackSave"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["stackSave"].apply(null, arguments)
};
var dynCall_ii = Module["dynCall_ii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_ii"].apply(null, arguments)
};
var dynCall_iiii = Module["dynCall_iiii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_iiii"].apply(null, arguments)
};
var dynCall_vi = Module["dynCall_vi"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_vi"].apply(null, arguments)
};
var dynCall_viii = Module["dynCall_viii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_viii"].apply(null, arguments)
};
var dynCall_viiii = Module["dynCall_viiii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_viiii"].apply(null, arguments)
};
var dynCall_viiiii = Module["dynCall_viiiii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_viiiii"].apply(null, arguments)
};
var dynCall_viiiiii = Module["dynCall_viiiiii"] = function () {
    assert(runtimeInitialized, 'you need to wait for the runtime to be ready (e.g. wait for main() to be called)');
    assert(!runtimeExited, 'the runtime was exited (use NO_EXIT_RUNTIME to keep it alive after main() exits)');
    return Module["asm"]["dynCall_viiiiii"].apply(null, arguments)
};
;


// === Auto-generated postamble setup entry stuff ===

Module['asm'] = asm;

if (!Module["intArrayFromString"]) Module["intArrayFromString"] = function () {
    abort("'intArrayFromString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["intArrayToString"]) Module["intArrayToString"] = function () {
    abort("'intArrayToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["ccall"]) Module["ccall"] = function () {
    abort("'ccall' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["cwrap"]) Module["cwrap"] = function () {
    abort("'cwrap' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["setValue"]) Module["setValue"] = function () {
    abort("'setValue' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getValue"]) Module["getValue"] = function () {
    abort("'getValue' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["allocate"]) Module["allocate"] = function () {
    abort("'allocate' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getMemory"]) Module["getMemory"] = function () {
    abort("'getMemory' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["Pointer_stringify"]) Module["Pointer_stringify"] = function () {
    abort("'Pointer_stringify' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["AsciiToString"]) Module["AsciiToString"] = function () {
    abort("'AsciiToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stringToAscii"]) Module["stringToAscii"] = function () {
    abort("'stringToAscii' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["UTF8ArrayToString"]) Module["UTF8ArrayToString"] = function () {
    abort("'UTF8ArrayToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["UTF8ToString"]) Module["UTF8ToString"] = function () {
    abort("'UTF8ToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stringToUTF8Array"]) Module["stringToUTF8Array"] = function () {
    abort("'stringToUTF8Array' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stringToUTF8"]) Module["stringToUTF8"] = function () {
    abort("'stringToUTF8' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["lengthBytesUTF8"]) Module["lengthBytesUTF8"] = function () {
    abort("'lengthBytesUTF8' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["UTF16ToString"]) Module["UTF16ToString"] = function () {
    abort("'UTF16ToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stringToUTF16"]) Module["stringToUTF16"] = function () {
    abort("'stringToUTF16' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["lengthBytesUTF16"]) Module["lengthBytesUTF16"] = function () {
    abort("'lengthBytesUTF16' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["UTF32ToString"]) Module["UTF32ToString"] = function () {
    abort("'UTF32ToString' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stringToUTF32"]) Module["stringToUTF32"] = function () {
    abort("'stringToUTF32' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["lengthBytesUTF32"]) Module["lengthBytesUTF32"] = function () {
    abort("'lengthBytesUTF32' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["allocateUTF8"]) Module["allocateUTF8"] = function () {
    abort("'allocateUTF8' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stackTrace"]) Module["stackTrace"] = function () {
    abort("'stackTrace' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addOnPreRun"]) Module["addOnPreRun"] = function () {
    abort("'addOnPreRun' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addOnInit"]) Module["addOnInit"] = function () {
    abort("'addOnInit' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addOnPreMain"]) Module["addOnPreMain"] = function () {
    abort("'addOnPreMain' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addOnExit"]) Module["addOnExit"] = function () {
    abort("'addOnExit' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addOnPostRun"]) Module["addOnPostRun"] = function () {
    abort("'addOnPostRun' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["writeStringToMemory"]) Module["writeStringToMemory"] = function () {
    abort("'writeStringToMemory' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["writeArrayToMemory"]) Module["writeArrayToMemory"] = function () {
    abort("'writeArrayToMemory' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["writeAsciiToMemory"]) Module["writeAsciiToMemory"] = function () {
    abort("'writeAsciiToMemory' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addRunDependency"]) Module["addRunDependency"] = function () {
    abort("'addRunDependency' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["removeRunDependency"]) Module["removeRunDependency"] = function () {
    abort("'removeRunDependency' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["ENV"]) Module["ENV"] = function () {
    abort("'ENV' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["FS"]) Module["FS"] = function () {
    abort("'FS' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["FS_createFolder"]) Module["FS_createFolder"] = function () {
    abort("'FS_createFolder' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createPath"]) Module["FS_createPath"] = function () {
    abort("'FS_createPath' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createDataFile"]) Module["FS_createDataFile"] = function () {
    abort("'FS_createDataFile' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createPreloadedFile"]) Module["FS_createPreloadedFile"] = function () {
    abort("'FS_createPreloadedFile' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createLazyFile"]) Module["FS_createLazyFile"] = function () {
    abort("'FS_createLazyFile' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createLink"]) Module["FS_createLink"] = function () {
    abort("'FS_createLink' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_createDevice"]) Module["FS_createDevice"] = function () {
    abort("'FS_createDevice' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["FS_unlink"]) Module["FS_unlink"] = function () {
    abort("'FS_unlink' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ). Alternatively, forcing filesystem support (-s FORCE_FILESYSTEM=1) can export this for you")
};
if (!Module["GL"]) Module["GL"] = function () {
    abort("'GL' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["staticAlloc"]) Module["staticAlloc"] = function () {
    abort("'staticAlloc' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["dynamicAlloc"]) Module["dynamicAlloc"] = function () {
    abort("'dynamicAlloc' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["warnOnce"]) Module["warnOnce"] = function () {
    abort("'warnOnce' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["loadDynamicLibrary"]) Module["loadDynamicLibrary"] = function () {
    abort("'loadDynamicLibrary' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["loadWebAssemblyModule"]) Module["loadWebAssemblyModule"] = function () {
    abort("'loadWebAssemblyModule' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getLEB"]) Module["getLEB"] = function () {
    abort("'getLEB' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getFunctionTables"]) Module["getFunctionTables"] = function () {
    abort("'getFunctionTables' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["alignFunctionTables"]) Module["alignFunctionTables"] = function () {
    abort("'alignFunctionTables' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["registerFunctions"]) Module["registerFunctions"] = function () {
    abort("'registerFunctions' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["addFunction"]) Module["addFunction"] = function () {
    abort("'addFunction' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["removeFunction"]) Module["removeFunction"] = function () {
    abort("'removeFunction' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getFuncWrapper"]) Module["getFuncWrapper"] = function () {
    abort("'getFuncWrapper' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["prettyPrint"]) Module["prettyPrint"] = function () {
    abort("'prettyPrint' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["makeBigInt"]) Module["makeBigInt"] = function () {
    abort("'makeBigInt' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["dynCall"]) Module["dynCall"] = function () {
    abort("'dynCall' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["getCompilerSetting"]) Module["getCompilerSetting"] = function () {
    abort("'getCompilerSetting' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stackSave"]) Module["stackSave"] = function () {
    abort("'stackSave' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stackRestore"]) Module["stackRestore"] = function () {
    abort("'stackRestore' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["stackAlloc"]) Module["stackAlloc"] = function () {
    abort("'stackAlloc' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["establishStackSpace"]) Module["establishStackSpace"] = function () {
    abort("'establishStackSpace' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["print"]) Module["print"] = function () {
    abort("'print' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["printErr"]) Module["printErr"] = function () {
    abort("'printErr' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
};
if (!Module["ALLOC_NORMAL"]) Object.defineProperty(Module, "ALLOC_NORMAL", {
    get: function () {
        abort("'ALLOC_NORMAL' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
    }
});
if (!Module["ALLOC_STACK"]) Object.defineProperty(Module, "ALLOC_STACK", {
    get: function () {
        abort("'ALLOC_STACK' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
    }
});
if (!Module["ALLOC_STATIC"]) Object.defineProperty(Module, "ALLOC_STATIC", {
    get: function () {
        abort("'ALLOC_STATIC' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
    }
});
if (!Module["ALLOC_DYNAMIC"]) Object.defineProperty(Module, "ALLOC_DYNAMIC", {
    get: function () {
        abort("'ALLOC_DYNAMIC' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
    }
});
if (!Module["ALLOC_NONE"]) Object.defineProperty(Module, "ALLOC_NONE", {
    get: function () {
        abort("'ALLOC_NONE' was not exported. add it to EXTRA_EXPORTED_RUNTIME_METHODS (see the FAQ)")
    }
});


/**
 * @constructor
 * @extends {Error}
 * @this {ExitStatus}
 */
function ExitStatus(status) {
    this.name = "ExitStatus";
    this.message = "Program terminated with exit(" + status + ")";
    this.status = status;
};
ExitStatus.prototype = new Error();
ExitStatus.prototype.constructor = ExitStatus;

var initialStackTop;
var calledMain = false;

dependenciesFulfilled = function runCaller() {
    // If run has never been called, and we should call run (INVOKE_RUN is true, and Module.noInitialRun is not false)
    if (!Module['calledRun']) run();
    if (!Module['calledRun']) dependenciesFulfilled = runCaller; // try this again later, after new deps are fulfilled
}

Module['callMain'] = function callMain(args) {
    assert(runDependencies == 0, 'cannot call main when async dependencies remain! (listen on __ATMAIN__)');
    assert(__ATPRERUN__.length == 0, 'cannot call main when preRun functions remain to be called');

    args = args || [];

    ensureInitRuntime();

    var argc = args.length + 1;
    var argv = stackAlloc((argc + 1) * 4);
    HEAP32[argv >> 2] = allocateUTF8OnStack(Module['thisProgram']);
    for (var i = 1; i < argc; i++) {
        HEAP32[(argv >> 2) + i] = allocateUTF8OnStack(args[i - 1]);
    }
    HEAP32[(argv >> 2) + argc] = 0;


    try {

        var ret = Module['_main'](argc, argv, 0);


        // if we're not running an evented main loop, it's time to exit
        exit(ret, /* implicit = */ true);
    }
    catch (e) {
        if (e instanceof ExitStatus) {
            // exit() throws this once it's done to make sure execution
            // has been stopped completely
            return;
        } else if (e == 'SimulateInfiniteLoop') {
            // running an evented main loop, don't immediately exit
            Module['noExitRuntime'] = true;
            return;
        } else {
            var toLog = e;
            if (e && typeof e === 'object' && e.stack) {
                toLog = [e, e.stack];
            }
            err('exception thrown: ' + toLog);
            Module['quit'](1, e);
        }
    } finally {
        calledMain = true;
    }
}


/** @type {function(Array=)} */
function run(args) {
    args = args || Module['arguments'];

    if (runDependencies > 0) {
        return;
    }

    writeStackCookie();

    preRun();

    if (runDependencies > 0) return; // a preRun added a dependency, run will be called later
    if (Module['calledRun']) return; // run may have just been called through dependencies being fulfilled just in this very frame

    function doRun() {
        if (Module['calledRun']) return; // run may have just been called while the async setStatus time below was happening
        Module['calledRun'] = true;

        if (ABORT) return;

        ensureInitRuntime();

        preMain();

        if (Module['onRuntimeInitialized']) Module['onRuntimeInitialized']();

        if (Module['_main'] && shouldRunNow) Module['callMain'](args);

        postRun();
    }

    if (Module['setStatus']) {
        Module['setStatus']('Running...');
        setTimeout(function () {
            setTimeout(function () {
                Module['setStatus']('');
            }, 1);
            doRun();
        }, 1);
    } else {
        doRun();
    }
    checkStackCookie();
}

Module['run'] = run;

function checkUnflushedContent() {
    // Compiler settings do not allow exiting the runtime, so flushing
    // the streams is not possible. but in ASSERTIONS mode we check
    // if there was something to flush, and if so tell the user they
    // should request that the runtime be exitable.
    // Normally we would not even include flush() at all, but in ASSERTIONS
    // builds we do so just for this check, and here we see if there is any
    // content to flush, that is, we check if there would have been
    // something a non-ASSERTIONS build would have not seen.
    // How we flush the streams depends on whether we are in NO_FILESYSTEM
    // mode (which has its own special function for this; otherwise, all
    // the code is inside libc)
    var print = out;
    var printErr = err;
    var has = false;
    out = err = function (x) {
        has = true;
    }
    try { // it doesn't matter if it fails
        var flush = flush_NO_FILESYSTEM;
        if (flush) flush(0);
    } catch (e) {
    }
    out = print;
    err = printErr;
    if (has) {
        warnOnce('stdio streams had content in them that was not flushed. you should set NO_EXIT_RUNTIME to 0 (see the FAQ), or make sure to emit a newline when you printf etc.');
    }
}

function exit(status, implicit) {
    checkUnflushedContent();

    // if this is just main exit-ing implicitly, and the status is 0, then we
    // don't need to do anything here and can just leave. if the status is
    // non-zero, though, then we need to report it.
    // (we may have warned about this earlier, if a situation justifies doing so)
    if (implicit && Module['noExitRuntime'] && status === 0) {
        return;
    }

    if (Module['noExitRuntime']) {
        // if exit() was called, we may warn the user if the runtime isn't actually being shut down
        if (!implicit) {
            err('exit(' + status + ') called, but NO_EXIT_RUNTIME is set, so halting execution but not exiting the runtime or preventing further async execution (build with NO_EXIT_RUNTIME=0, if you want a true shutdown)');
        }
    } else {

        ABORT = true;
        EXITSTATUS = status;
        STACKTOP = initialStackTop;

        exitRuntime();

        if (Module['onExit']) Module['onExit'](status);
    }

    Module['quit'](status, new ExitStatus(status));
}

var abortDecorators = [];

function abort(what) {
    if (Module['onAbort']) {
        Module['onAbort'](what);
    }

    if (what !== undefined) {
        out(what);
        err(what);
        what = JSON.stringify(what)
    } else {
        what = '';
    }

    ABORT = true;
    EXITSTATUS = 1;

    var extra = '';
    var output = 'abort(' + what + ') at ' + stackTrace() + extra;
    if (abortDecorators) {
        abortDecorators.forEach(function (decorator) {
            output = decorator(output, what);
        });
    }
    throw output;
}

Module['abort'] = abort;

// {{PRE_RUN_ADDITIONS}}

if (Module['preInit']) {
    if (typeof Module['preInit'] == 'function') Module['preInit'] = [Module['preInit']];
    while (Module['preInit'].length > 0) {
        Module['preInit'].pop()();
    }
}

// shouldRunNow refers to calling main(), not run().
var shouldRunNow = true;
if (Module['noInitialRun']) {
    shouldRunNow = false;
}

Module["noExitRuntime"] = true;

run();

// {{POST_RUN_ADDITIONS}}


// {{MODULE_ADDITIONS}}



