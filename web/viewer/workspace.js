"use strict";

// Go's js/wasm syscall layer expects the Node callback-style filesystem API.
// This per-tab filesystem holds only files imported or generated in the editor.
(() => {
  const root = "/workspace", maxBytes = 128 * 1024 * 1024;
  const flags = {O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024, O_DIRECTORY: 65536};
  const paths = new Map(), handles = new Map();
  const decoder = new TextDecoder();
  let nextFD = 3, nextInode = 1, cwd = root, total = 0, serial = 0;
  function fail(code, path) {
    throw Object.assign(new Error(path ? `${code}: ${path}` : code), {code});
  }
  function resolve(...parts) {
    let current = cwd.split("/").filter(Boolean);
    for (const value of parts) {
      if (value.startsWith("/")) current = [];
      for (const part of value.split("/")) {
        if (part === "..") current.pop();
        else if (part && part !== ".") current.push(part);
      }
    }
    return "/" + current.join("/");
  }
  const parent = p => p.slice(0, p.lastIndexOf("/")) || "/";
  function entry(directory, permissions) {
    return {directory, permissions, data: new Uint8Array(), size: 0, time: Date.now(), version: ++serial, ino: nextInode++};
  }
  paths.set("/", entry(true, 0o755));
  paths.set(root, entry(true, 0o755));
  paths.set("/tmp", entry(true, 0o700));
  function lookup(p) {
    const n = paths.get(p);
    if (!n) fail("ENOENT", p);
    return n;
  }
  function directory(p) {
    if (!lookup(p).directory) fail("ENOTDIR", p);
  }
  function children(p) {
    directory(p);
    return Array.from(paths.keys()).filter(k => k !== p && parent(k) === p).map(k => k.slice(k.lastIndexOf("/") + 1));
  }
  function fd(id) {
    const handle = handles.get(id);
    if (!handle) fail("EBADF");
    return handle;
  }
  function resize(n, size) {
    if (!Number.isSafeInteger(size) || size < 0) fail("EINVAL");
    if (total + size - n.size > maxBytes) fail("ENOSPC");
    if (size > n.data.length) {
      const next = new Uint8Array(Math.max(size, Math.min(maxBytes, Math.max(256, n.data.length * 2))));
      next.set(n.data.subarray(0, n.size));
      n.data = next;
    } else if (size < n.size) {
      n.data.fill(0, size, n.size);
      if (size < n.data.length / 4) n.data = n.data.slice(0, size);
    }
    total += size - n.size;
    n.size = size;
    n.time = Date.now();
    n.version = ++serial;
  }
  function stat(n) {
    return {dev: 0, ino: n.ino, mode: (n.directory ? 0o40000 : 0o100000) | n.permissions, nlink: 1,
      uid: 0, gid: 0, rdev: 0, size: n.size, blksize: 4096, blocks: Math.ceil(n.size / 512),
      atimeMs: n.time, mtimeMs: n.time, ctimeMs: n.time, isDirectory: () => n.directory};
  }
  function open(name, mode, permissions) {
    const p = resolve(name);
    let n = paths.get(p);
    if (!n) {
      if (!(mode & flags.O_CREAT)) fail("ENOENT", p);
      directory(parent(p));
      n = entry(false, permissions);
      paths.set(p, n);
    } else if ((mode & flags.O_CREAT) && (mode & flags.O_EXCL)) fail("EEXIST", p);
    if (mode & flags.O_DIRECTORY && !n.directory) fail("ENOTDIR", p);
    if (n.directory && mode & (flags.O_RDWR | flags.O_WRONLY)) fail("EISDIR", p);
    if (mode & flags.O_TRUNC && !n.directory) resize(n, 0);
    const id = nextFD++;
    handles.set(id, {n, mode, position: 0});
    return id;
  }
  function writeSync(id, data, offset = 0, length = data.length, position = null) {
    if (id === 1 || id === 2) {
      (id === 1 ? console.log : console.error)(decoder.decode(data.subarray(offset, offset + length)).trimEnd());
      return length;
    }
    const h = fd(id), n = h.n;
    if (!(h.mode & (flags.O_WRONLY | flags.O_RDWR))) fail("EBADF");
    const start = h.mode & flags.O_APPEND ? n.size : position ?? h.position;
    if (start + length > n.size) resize(n, start + length);
    n.data.set(data.subarray(offset, offset + length), start);
    n.time = Date.now();
    n.version = ++serial;
    if (position == null) h.position = start + length;
    return length;
  }
  function remove(p, isDir) {
    const n = lookup(p);
    if (n.directory !== isDir) fail(isDir ? "ENOTDIR" : "EISDIR", p);
    if (isDir && (p === "/" || p === root || children(p).length)) fail("ENOTEMPTY", p);
    paths.delete(p);
    if (!Array.from(handles.values()).some(h => h.n === n)) total -= n.size;
  }
  function close(id) {
    const n = fd(id).n;
    handles.delete(id);
    if (!Array.from(paths.values()).includes(n) && !Array.from(handles.values()).some(h => h.n === n)) total -= n.size;
  }
  const sync = {
    open, close, write: writeSync,
    read(id, data, offset, length, position) {
      const h = fd(id);
      if (h.n.directory) fail("EISDIR");
      if (h.mode & flags.O_WRONLY) fail("EBADF");
      const start = position ?? h.position, count = Math.max(0, Math.min(length, h.n.size - start));
      data.set(h.n.data.subarray(start, start + count), offset);
      if (position == null) h.position += count;
      return count;
    },
    fstat: id => stat(fd(id).n), stat: p => stat(lookup(resolve(p))), lstat: p => stat(lookup(resolve(p))),
    readdir: p => children(resolve(p)),
    mkdir(name, permissions) {
      const p = resolve(name);
      if (paths.has(p)) fail("EEXIST", p);
      directory(parent(p));
      paths.set(p, entry(true, permissions));
    },
    unlink: name => remove(resolve(name), false),
    rmdir: name => remove(resolve(name), true),
    rename(from, to) {
      const a = resolve(from), b = resolve(to), n = lookup(a);
      if (a === b) return;
      if (a === "/" || a === root || b.startsWith(a + "/")) fail("EINVAL", a);
      directory(parent(b));
      if (paths.has(b)) remove(b, n.directory);
      for (const [key, node] of Array.from(paths)) {
        if (key !== a && !key.startsWith(a + "/")) continue;
        paths.delete(key); paths.set(b + key.slice(a.length), node); node.version = ++serial;
      }
    },
    truncate: (p, size) => resize(lookup(resolve(p)), size),
    ftruncate: (id, size) => resize(fd(id).n, size),
    chmod: (p, mode) => { lookup(resolve(p)).permissions = mode & 0o777; },
    fchmod: (id, mode) => { fd(id).n.permissions = mode & 0o777; },
    utimes: (p, atime, mtime) => { lookup(resolve(p)).time = mtime * 1000; },
    fsync: id => { fd(id); },
    readlink: p => { lookup(resolve(p)); fail("EINVAL", p); }
  };
  for (const name of ["chown", "fchown", "lchown", "link", "symlink"]) sync[name] = () => fail("ENOSYS");
  const fs = {constants: flags, writeSync};
  for (const [name, operation] of Object.entries(sync)) {
    fs[name] = (...args) => {
      const callback = args.pop();
      let result;
      try { result = operation(...args); }
      catch (error) { callback(error); return; }
      callback(null, result);
    };
  }
  globalThis.fs = fs;
  globalThis.path = {resolve};
  globalThis.process ??= {pid: 1, ppid: 0, getuid: () => 0, getgid: () => 0, geteuid: () => 0, getegid: () => 0, getgroups: () => [], umask: () => 0};
  globalThis.process.cwd = () => cwd;
  globalThis.process.chdir = name => { const p = resolve(name); directory(p); cwd = p; };
  globalThis.stWorkspace = {
    put(name, data) {
      if (!name || name === "." || name === ".." || /[/\\\x00-\x1f]/.test(name)) fail("EINVAL", name);
      const id = open(root + "/" + name, flags.O_CREAT | flags.O_WRONLY, 0o644);
      try { resize(fd(id).n, data.length); writeSync(id, data); } finally { close(id); }
    },
    writeText(name, text) {
      const p = resolve(root, name);
      if (!p.startsWith(root + "/")) fail("EINVAL", name);
      const data = new TextEncoder().encode(text);
      const id = open(p, flags.O_CREAT | flags.O_WRONLY, 0o644);
      try { resize(fd(id).n, data.length); writeSync(id, data); } finally { close(id); }
    },
    list() {
      return Array.from(paths).filter(([p, n]) => !n.directory && p.startsWith(root + "/"))
        .map(([p, n]) => ({path: p.slice(root.length + 1), size: n.size})).sort((a, b) => a.path.localeCompare(b.path));
    },
    entries() {
      return Array.from(paths)
        .filter(([path]) => path.startsWith(root + "/"))
        .map(([path, n]) => ({path: path.slice(root.length + 1), directory: n.directory, version: n.version}));
    },
    apply(files, removed = []) {
      // Validate capacity and paths before replacing any committed file.
      let size = total;
      const checked = name => {
        const p = resolve(root, name);
        if (!p.startsWith(root + "/") || p.slice(root.length + 1) !== name) fail("EINVAL", name);
        return p;
      };
      for (const path of removed) size -= paths.get(checked(path))?.size || 0;
      for (const file of files) {
        size += (file.data?.length || 0) - (paths.get(checked(file.path))?.size || 0);
      }
      if (size > maxBytes) fail("ENOSPC");
      for (const path of removed) {
        const p = checked(path);
        const n = paths.get(p);
        if (n) { paths.delete(p); total -= n.size; }
      }
      for (const file of files) {
        const p = checked(file.path);
        const old = paths.get(p);
        if (old?.directory && file.directory) continue;
        const n = entry(file.directory, file.directory ? 0o755 : 0o644);
        if (!file.directory) {
          n.data = file.data;
          n.size = file.data.length;
        }
        paths.set(p, n);
        total += n.size - (old?.size || 0);
      }
    },
    get(name) {
      const p = resolve(root, name), n = lookup(p);
      if (!p.startsWith(root + "/") || n.directory) fail("EINVAL", name);
      return n.data.slice(0, n.size);
    }
  };
})();
