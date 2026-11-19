#!/usr/bin/env python3
"""Add the exact Natro Editor Windows icon resources to an unsigned Go PE32+ EXE.

The .rsrc payload comes from the supplied desktop app. No code is downloaded;
Python standard library only. This script adds a normal PE resource section,
updates the resource data RVAs, and fixes the PE headers.
"""
from pathlib import Path
import argparse
import struct

ORIGINAL_RSRC_RVA = 0x906000  # from the exact Natro_Editor_Desktop_Name_Fix.exe

def u16(data, pos): return struct.unpack_from('<H', data, pos)[0]
def u32(data, pos): return struct.unpack_from('<I', data, pos)[0]
def p16(data, pos, value): struct.pack_into('<H', data, pos, value)
def p32(data, pos, value): struct.pack_into('<I', data, pos, value)
def align(value, multiple): return (value+multiple-1)//multiple*multiple

def patch_resource_data_rvas(resource_bytes, diff):
    out = bytearray(resource_bytes)
    visited_dirs = set()
    visited_entries = set()
    def walk(at):
        if at in visited_dirs: return
        visited_dirs.add(at)
        if at+16 > len(out): raise ValueError('invalid resource directory')
        num = u16(out, at+12)+u16(out, at+14)
        for i in range(num):
            ent = at+16+i*8
            if ent+8 > len(out): raise ValueError('invalid resource entry')
            dest = u32(out, ent+4)
            rel = dest & 0x7fffffff
            if dest & 0x80000000:
                walk(rel)
            elif rel not in visited_entries:
                if rel+16 > len(out): raise ValueError('invalid resource data entry')
                old = u32(out, rel)
                if old < ORIGINAL_RSRC_RVA or old >= ORIGINAL_RSRC_RVA+len(out):
                    raise ValueError('resource payload points outside template')
                p32(out, rel, old+diff)
                visited_entries.add(rel)
    walk(0)
    if not visited_entries: raise ValueError('resource data entries not found')
    return out

def add_resource(exe, source):
    data = bytearray(exe)
    template = bytes(source)
    if data[:2]!=b'MZ': raise ValueError('not an MZ executable')
    peoff=u32(data,0x3c)
    if data[peoff:peoff+4]!=b'PE\0\0': raise ValueError('not a PE executable')
    machine=u16(data,peoff+4)
    if machine!=0x8664: raise ValueError('requires x64 executable')
    n=u16(data,peoff+6)
    optional=u16(data,peoff+20)
    opt=peoff+24
    if u16(data,opt)!=0x20b: raise ValueError('requires PE32+ executable')
    sect=opt+optional
    section_align=u32(data,opt+32)
    file_align=u32(data,opt+36)
    if file_align == 0 or section_align == 0: raise ValueError('bad PE alignment')
    head_end=sect+(n+1)*40
    first_raw = len(data)
    max_virt=0
    for i in range(n):
        q=sect+i*40
        name=bytes(data[q:q+8]).rstrip(b'\0')
        if name==b'.rsrc': raise ValueError('EXE already contains a resource section')
        vsize=u32(data,q+8)
        vaddr=u32(data,q+12)
        rawsize=u32(data,q+16)
        rawptr=u32(data,q+20)
        if rawsize and rawptr: first_raw=min(first_raw,rawptr)
        max_virt=max(max_virt,vaddr+max(vsize,rawsize))
    if head_end > first_raw:
        raise ValueError('not enough PE header space for resource section')
    new_rva=align(max_virt,section_align)
    new_offset=align(len(data),file_align)
    adjusted=patch_resource_data_rvas(template,new_rva-ORIGINAL_RSRC_RVA)
    rawsize=align(len(adjusted),file_align)
    data+=b'\0'*(new_offset-len(data))
    data+=adjusted
    data+=b'\0'*(rawsize-len(adjusted))
    q=sect+n*40
    data[q:q+40]=struct.pack('<8sIIIIIIHHI',b'.rsrc\0\0\0',len(adjusted),new_rva,rawsize,new_offset,0,0,0,0,0x40000040)
    p16(data,peoff+6,n+1)
    p32(data,opt+8,u32(data,opt+8)+rawsize)  # SizeOfInitializedData
    p32(data,opt+56,align(new_rva+len(adjusted),section_align))  # SizeOfImage
    p32(data,opt+64,0)  # checksum (optional for normal user-mode EXEs)
    p32(data,opt+112+2*8,new_rva)  # DataDirectory[RESOURCE].VirtualAddress
    p32(data,opt+112+2*8+4,len(adjusted))  # DataDirectory[RESOURCE].Size
    return data

if __name__ == '__main__':
    cli=argparse.ArgumentParser(description=__doc__)
    cli.add_argument('input_exe',type=Path)
    cli.add_argument('output_exe',type=Path)
    cli.add_argument('resource_section',type=Path)
    args=cli.parse_args()
    if args.input_exe.resolve()==args.output_exe.resolve():
        cli.error('output must be a different filename')
    result=add_resource(args.input_exe.read_bytes(),args.resource_section.read_bytes())
    args.output_exe.parent.mkdir(parents=True,exist_ok=True)
    args.output_exe.write_bytes(result)
    print(f'Added icon resource: {args.output_exe} ({len(result):,} bytes)')
