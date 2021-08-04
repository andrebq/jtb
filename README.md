# jtb
JSON/Javascript Toolbox

**jtb** is a simple cli app and a library to provide a simple way to add scripting capabilities to
applications using Javascript (namely the goja javascript engine).

**jtb** does not aim to be a replacement for the NodeJS/NPM ecosystem, instead,
it aims to be a middle ground between `jq` and `node`, one of the main goals
is to simply the process of importing snippets of code from external sources,
without exposing too much of the underlying system.

## Motivation for jtb

More often than not, `jq` is used in conjunction with bash to execute non-trivial tasks. Doing
tree manipulation in bash is `tricky` to say the least. **jtb** was created out of my frustration
while dealing with Yaml/JSON/Bash/cURL/kubectl to perform, trival tree
operations (put node, replace node, delete node, save to disk).
