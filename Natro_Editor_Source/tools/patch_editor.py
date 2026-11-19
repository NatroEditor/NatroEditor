#!/usr/bin/env python3
"""Optional frontend patcher for the supplied compiled desktop core.

The compiled Go core's source is not available. Its embedded HTML can be replaced
only when the new UTF-8 file fits the *same byte span*. This lets small content,
CSS and JavaScript changes survive without making misleading claims about
recompiling the original native backend. Always back up before changing.
"""
from pathlib import Path
import argparse

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--base-core',type=Path,default=Path('launcher/Natro_Editor_core.exe'))
parser.add_argument('--html',type=Path,default=Path('editor/Natro_Editor.html'))
parser.add_argument('--output-core',type=Path,default=Path('launcher/Natro_Editor_core_patched.exe'))
args=parser.parse_args()
core=args.base_core.read_bytes()
start=core.index(b'<!DOCTYPE html>')
end=core.index(b'</html>',start)+len(b'</html>')
original_length=end-start
html=args.html.read_bytes()
if len(html)>original_length:
    parser.error(f'HTML is {len(html)-original_length} bytes too long. This native Go core has a fixed embedding span.')
if len(html)<original_length:
    # Whitespace within <style> does not change the editor's behavior.
    string=html.decode('utf-8')
    pad=original_length-len(html)
    if '</style>' not in string:parser.error('Cannot find </style> to pad HTML')
    string=string.replace('</style>', ' '*pad+'</style>',1)
    html=string.encode('utf-8')
assert len(html)==original_length
args.output_core.parent.mkdir(parents=True,exist_ok=True)
args.output_core.write_bytes(core[:start]+html+core[end:])
print('Patched compiled native core:',args.output_core)
print('To use it, replace launcher/Natro_Editor_core.exe with the patched copy and rebuild the launcher.')
