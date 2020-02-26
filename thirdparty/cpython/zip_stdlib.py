from __future__ import print_function

import calendar
import io
import marshal
import os
import py_compile
import struct
import sys
import tempfile
import zipfile

PY3 = sys.version_info[0] >= 3
if not PY3:
    import imp

EPOCH = (2018, 11, 11, 11, 11, 11)
EPOCH_BIN = struct.pack('<L', calendar.timegm(EPOCH))

def main():
    mode = sys.argv[1]
    assert mode in ("sloppy", "final")
    out = sys.argv[2]
    with zipfile.ZipFile(out, "w") as z:
        for inp in sys.argv[3:]:
            _, l, arcname = inp.partition("/Lib/")
            if not l:
                print("unrecognized filename %r" % inp, file=sys.stderr)
                sys.exit(1)
            if mode == 'final':
                info = zipfile.ZipInfo(arcname, EPOCH)
                with open(inp, 'rb') as fp:
                    source = fp.read()
                    z.writestr(info, source)
                    if inp.endswith(".py"):
                        if PY3:
                            tf = tempfile.NamedTemporaryFile()
                            tf.close()
                            py_compile.compile(
                                inp,
                                cfile=tf.name,
                                dfile=arcname,
                                invalidation_mode=py_compile.PycInvalidationMode.UNCHECKED_HASH,
                            )
                            with open(tf.name, "rb") as tfh:
                                # zipped stdlib import doesn't use the __pycache__ paths
                                info = zipfile.ZipInfo(arcname + "c", EPOCH)
                                z.writestr(info, tfh.read())
                        else:
                            code = compile(source, arcname, 'exec', dont_inherit=True)
                            data = io.BytesIO()
                            data.write(imp.get_magic())
                            data.write(EPOCH_BIN)
                            data.write(marshal.dumps(code))
                            info = zipfile.ZipInfo(arcname + "c", EPOCH)
                            z.writestr(info, data.getvalue())
            else:
                z.write(inp, arcname)

main()
